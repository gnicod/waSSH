package main

/*
TODO
-pouvoir passer le port en param ( -p --port)
-pouvoir passer un script en param qui recup la liste des servers ( --server-from-script )
-pouvoir rediriger la sortie vers un fichier different / server eg ( --exploded-output="/tmp/%SERVER%_res" )
-better script https://stackoverflow.com/questions/20437336/how-to-execute-system-command-in-golang-with-unknown-arguments
DONE
-pouvoir passer une liste de commande en stdin
-pouvoir passer une cle en param ( -i )
-pouvoir passer le user en param ( -u --user)
-pouvoir mettre port et user dans le server
*/

import (
	"fmt"
	"github.com/droundy/goopt"
	"github.com/tsuru/config"
	"github.com/andrew-d/go-termutil"
	"os/exec"
	"log"
	"strings"
	"os"
	"bufio"
	"runtime"
	"time"
	"strconv"
	"regexp"
)

//Command-line flag
var group = goopt.String([]string{"-g", "--group"}, "default", "name of group")
var command = goopt.String([]string{"-c", "--command"}, "", "predefined command to execute")
var execute = goopt.String([]string{"-e", "--execute"}, "", "command to execute")
var user = goopt.String([]string{"-u", "--user"}, "", "name of user")
var port = goopt.String([]string{"-p", "--port"}, "", "port")
var sTimeout = goopt.String([]string{"-t", "--timeout"}, "", "timeout (second) before leaving")
var key = goopt.String([]string{"-i", "--key"}, "", " Selects a file from which the identity (private key) for RSA or DSA authentication is read")
var showlist = goopt.Flag([]string{"-l", "--list"}, []string{}, "list all commands available", "")

type Server struct {
	user , hostname , port string
}

func getStdin() (s []string) {
	stdin := bufio.NewScanner(os.Stdin)
	for stdin.Scan() {
		line := stdin.Bytes()
		s = append(s,string(line))
	}
	return s
}

func showListCommand() {
	cmds, _ := config.Get("commands")
	m := cmds.(map[interface{}]interface{})
	fmt.Printf("Commands available: \n\n")
	for k, v := range m {
		cmd := v.(map[interface{}]interface{})
		fmt.Printf("%s : \n\t $> %s\n\t -%s\n\n", k, cmd["cmd"], cmd["desc"])

	}
	os.Exit(0)

}


func getDefaultValue(cnf string) string {
	defValue := make(map[string]string)
	defValue["user"] = "root"
	defValue["timeout"] = "12"
	defValue["port"] = "22"
	//defValue["key"] = getHomeDirectory()+"\\.ssh\\id_rsa" //TODO chercher id_dsa si pas de rsa
	if runtime.GOOS == "windows"{
		defValue["key"] = getHomeDirectory()+"\\.ssh\\id_rsa" //TODO chercher id_dsa si pas de rsa
	}else{
		defValue["key"] = getHomeDirectory() + "/.ssh/id_rsa" //TODO chercher id_dsa si pas de rsa
	}
	//TOFIX key key not found
	val, er := config.GetString(cnf)
	if er != nil {
		val = defValue[cnf]
	}
	return val
}

func getHomeDirectory() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")

		}
		return home
	}
	return os.Getenv("HOME")
}

func getConfigFile() string {
	home := getHomeDirectory()
	if runtime.GOOS == "windows" {
		return home + "\\_wasshrc"
	}
	return home + "/.wasshrc"
}

func parseLineServer(line string) (server Server){
	server.user = *user
	server.port = "22"
	r, _ := regexp.Compile("([a-z]*@)?([a-z0-9.]*):?([0-9]{1,4})?")
	matches := r.FindStringSubmatch(line)
	if len(matches[1])>0{
		server.user = matches[1][:len(matches[1])-1]
	}
	server.hostname = matches[2]
	if len(matches[3])>0{
		server.port = matches[3]
	}
	return server
}

func GetServers(group string) (servers []Server) {
	hosts, err := config.GetList("groups:" + group)
	for _, line := range hosts{
		server := parseLineServer(line)
		servers = append(servers,server)
	}
	if err != nil {
		grScript,errScr := config.GetString("groups:"+group)
		if errScr != nil {
			fmt.Printf("Group %s does not exists " , group)
			os.Exit(1)
		}
		outResScr, errResScr := exec.Command("sh","-c",grScript).Output() //TOFIX
		if errResScr != nil {
			log.Fatal(errResScr)
			fmt.Printf("Script %s does not exists " , grScript)
			os.Exit(1)
		}
		// TOFIX quite ugly
		hostsTmp := strings.Split(string(outResScr),"\n")
		for _, host := range hostsTmp {
			if len(host)>0{
				server := parseLineServer(host)
				servers = append(servers,server)
			}
		}
	}
	return servers
}

func ExecuteSsh(res chan string, server Server, commands []string) {
	client := NewSSHClient(server.hostname+":"+server.port, server.user, *key)
	//TODO split command \n
	fullOut := fmt.Sprintf("\033[92m%s :\033[0m \n",server.hostname)
	for _,cmd := range commands {
		out, err := client.Run(cmd)
		if err != nil {
			fmt.Println(err)
		}
		out = fmt.Sprintf("\033[31m%s\033[0m :\n%s", cmd, out)
		fullOut = fullOut + out
	}
	res <- fullOut
}

func main() {
	goopt.Description = func() string {
		return "The clean way to ssh'em all."
	}
	goopt.Version = "0.1"
	goopt.Summary = "the clean way to SSH'em all"
	goopt.Parse(nil)
	err := config.ReadConfigFile(getConfigFile())
	if err != nil {
		log.Fatal(err)
		log.Fatalf("%s doesn't exists or is not wellformed", getConfigFile())
	}
	if len(*user)==0{
		*user = getDefaultValue("user")
	}
	if len(*key)==0{
		*key = getDefaultValue("key")
	}
	if len(*port)==0{
		*port = getDefaultValue("port")
	}
	if len(*sTimeout)==0{
		*sTimeout = getDefaultValue("timeout")
	}
	if *showlist {
		showListCommand()
	}
	timeout, erParseTime := strconv.ParseInt(*sTimeout, 0, 64)
	if erParseTime != nil{
		fmt.Println(erParseTime)
	}
	hosts := GetServers(*group)

	var cmd []string
	if termutil.Isatty(os.Stdin.Fd()) {
		//stdin empty
		if len(*execute)>0{
			cmd = append(cmd,*execute)
		}
	}else{
		cmd = getStdin()
	}

if len(*command) > 0 {
	com, err := config.GetString("commands:" + *command + ":cmd")
	if err != nil {
		fmt.Printf("Command does not exists: %s\n", *command)
		os.Exit(1)
	}
	cmd = append(cmd,com)
}

	//TODO check  *command 

	//create the chan
	sshResultChan := make(chan string)

	for _, host := range hosts {
		go ExecuteSsh(sshResultChan, host, cmd)
	}
	for _ = range hosts {
		//Catch the result
		select {
		case res := <-sshResultChan:
			fmt.Println(res)
		case <-time.After(time.Second * time.Duration(timeout)):
			fmt.Println("timeout ")
		}
	}

}
