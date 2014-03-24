package main

/*
* TODO:
*
*	If config file does not exists, ask to create it
*
*   Commandes dispos:
*     execute, get, push
*			--sync synchronize conf file
*
*		Options:
*			-i to select a different private key
*			-c to select a predefined command in the config file
 */

import (
	"fmt"
	"github.com/droundy/goopt"
	"github.com/gnicod/goscplib"
	"github.com/tsuru/config"
	"os"
)

const (
	CONF_FILE = "config.yaml"
	//action
	ACTION_SSH        = 0
	ACTION_SCP        = 1
	ACTION_SYNC_GROUP = 2
	ACTION_SYNC_FILE  = 3
)

type FileSync struct {
	local, dest, post_cmd, group string
}

//Command-line flag
var group = goopt.String([]string{"-g", "--group"}, "default", "name of group")
var command = goopt.String([]string{"-c", "--command"}, "", "predefined command to execute")
var execute = goopt.String([]string{"-e", "--execute"}, "", "command to execute")
var user = goopt.String([]string{"-u", "--user"}, "root", "name of user")
var pwd = goopt.String([]string{"-p", "--password"}, "", "password")
var promptpwd = goopt.Flag([]string{"--prompt-pwd"}, []string{}, "prompt password", "")
var showlist = goopt.Flag([]string{"-l", "--list"}, []string{}, "list all commands available", "")
var silent = goopt.Flag([]string{"-s", "--silent"}, []string{}, "quiet mode", "")

//scp options
var src = goopt.String([]string{"--src"}, "", "source file to push on the remote server")
var dest = goopt.String([]string{"--dest"}, "", "destination where to push on the remote server")
var syncgroup = goopt.String([]string{"--sync-group"}, "", "group to synchronize")
var syncfile = goopt.String([]string{"--sync-file"}, "", "file to synchronize")

func executeSsh(res chan string, server string, command string) {
	client := NewSSHClient(server, *user)
	client.Connect()
	pemfile, _ := config.GetString("key") //TODO catch error
	client.PemFile = pemfile
	output := ""
	if *silent {
		output = client.Execute(command)
	} else {
		output = "\033[92m" + server + ":\033[0m \n" + client.Execute(command)
	}
	res <- output
}

func executeScp(res chan string, server string, src string, dest string) {
	client := NewSSHClient(server, *user)
	scp := goscplib.NewScp(client.Conn)
	fileSrc, srcErr := os.Open(src)
	if srcErr != nil {
		fmt.Println("Failed to open source file: " + srcErr.Error())
	}
	//Check if src is a dir
	srcStat, statErr := fileSrc.Stat()
	if statErr != nil {
		fmt.Println("Failed to stat file: " + statErr.Error())
	}
	if srcStat.IsDir() {
		scp.PushDir(src, dest)
	} else {
		scp.PushFile(src, dest)
	}
	if res != nil {
		res <- "\033[1m\033[92m" + server + ":\033[0m \n scp " + src + " to " + dest + "\n"
	}
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

func syncFile(name string, server string) string {
	files, _ := config.Get("files")
	mfiles := files.(map[interface{}]interface{})
	file := mfiles[name].(map[interface{}]interface{})
	fileSync := FileSync{local: file["local"].(string), dest: file["dest"].(string), post_cmd: file["post_cmd"].(string), group: file["group"].(string)}
	executeScp(nil, server, fileSync.local, fileSync.dest)
	return "SCP: " + fileSync.local + " on " + server + "\n"
}

func syncGroup(res chan string, group string, server string) {
	files, _ := config.Get("files")
	m := files.(map[interface{}]interface{})
	resul := ""
	for k, v := range m {
		file := v.(map[interface{}]interface{})
		if file["group"] == group {
			resul = resul + syncFile(k.(string), server)
		}
	}
	res <- resul

}

func GetServers(group string) []string {
	hosts, err := config.GetList("groups:" + group)
	if err != nil {
		fmt.Printf("Group does not exists: %s\n", group)
		os.Exit(1)
	}
	return hosts
}

func main() {
	goopt.Description = func() string {
		return "Manage server with ssh."
	}
	goopt.Version = "0.05"
	goopt.Summary = "one line to SSH'em all"
	goopt.Parse(nil)
	err := config.ReadConfigFile(CONF_FILE)
	hosts := GetServers(*group)

	if *showlist {
		showListCommand()
	}

	action := ACTION_SSH

	if len(*src) == 0 && len(*dest) > 0 {
		fmt.Println("--src should be setted")
		os.Exit(1)
	}
	if len(*src) > 0 && len(*dest) == 0 {
		fmt.Println("--dest should be setted")
		os.Exit(1)
	}

	if len(*src) > 0 && len(*dest) > 0 {
		action = ACTION_SCP
		if len(*command) > 0 {
			fmt.Println("The command flag will be ignored")
		}
		if len(*execute) > 0 {
			fmt.Println("The execute flag will be ignored")
		}
	}

	if len(*syncfile) > 0 {
		action = ACTION_SYNC_FILE
	}
	if len(*syncgroup) > 0 {
		action = ACTION_SYNC_GROUP
		hosts = GetServers(*syncgroup)
		*group = *syncgroup
	}

	cmd := *execute
	if len(*command) > 0 {
		cmd, err = config.GetString("commands:" + *command + ":cmd")
		if err != nil {
			fmt.Printf("Command does not exists: %s\n", *command)
			os.Exit(1)
		}
	}

	/*
		if len(*pwd) == 0  {
			*pwd, _ = getpass.GetPass()

			//var pass string
			//fmt.Print("Password: ")
			//fmt.Scanf("%s",&pass)
			//*pwd = pass
		}
	*/

	sshResultChan := make(chan string)
	for _, host := range hosts {
		switch action {
		case ACTION_SSH:
			go executeSsh(sshResultChan, host, cmd)

		case ACTION_SCP:
			go executeScp(sshResultChan, host, *src, *dest)

		case ACTION_SYNC_FILE:
			fmt.Println(sshResultChan, "sync file")
			os.Exit(1)

		case ACTION_SYNC_GROUP:
			go syncGroup(sshResultChan, *group, host)
			//os.Exit(1)
		}
	}

	if !*silent {
		fmt.Println("$", cmd, "\n")
	}
	for _ = range hosts {
		//Catch the result
		res := <-sshResultChan
		fmt.Println(res)
	}

	if err != nil {
		panic("Failed to dial: " + err.Error())
	}

}
