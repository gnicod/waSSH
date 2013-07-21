package main

/*
* TODO:
*
*	If config file does not exists, ask to create it
*
*   Commandes dispos:
*     execute, get, push
*			--sync synchronize conf file 
*			--check-sync check md5 
*
*		Options:
*			-i to select a different private key
*			-c to select a predefined command in the config file
*/

import (
	"github.com/gcmurphy/getpass"
	"github.com/globocom/config"
	"github.com/droundy/goopt"
	"fmt"
	"os"
)

const (
	CONF_FILE = "config.yaml"
)

//Command-line flag
var group     = goopt.String([]string{"-g", "--group"}, "default", "name of group")
var command   = goopt.String([]string{"-c", "--command"},"", "predefined command to execute")
var execute   = goopt.String([]string{"-e", "--execute"},"", "command to execute")
var user      = goopt.String([]string{"-u", "--user"}, "root", "name of user")
var pwd       = goopt.String([]string{"-p", "--password"}, "", "password")
var promptpwd = goopt.Flag([]string{"--prompt-pwd"}, []string{},"prompt password","")
var showlist  = goopt.Flag([]string{"-l", "--list"}, []string{},"list","")
//scp options
var src       = goopt.String([]string{"--src"}, "", "source file to push on the remote server")
var dest      = goopt.String([]string{"--dest"}, "", "destination where to push on the remote server")

func executeSsh(res chan string, server string, command string) {
	conn,_ := Connect(server, *user, *pwd)
	res <- "\033[1m\033[92m" + server + ":\033[0m \n" + Execute(conn, command) + "\n"
}

func executeScp(res chan string, server string, src string, dest string) {
	conn,_ := Connect(server, *user, *pwd)
	res <- "\033[1m\033[92m" + server + ":\033[0m \n" + Push(conn, src, dest) + "\n"
}

func showListCommand() {
	list, _ := config.Get("commands")
	//TODO
	fmt.Println(list)
	os.Exit(0)
}

func main() {
	goopt.Description = func() string {
		return "Example program for using the goopt flag library."

	}
	goopt.Version = "0.05"
	goopt.Summary = "one line to SSH'em all"
	goopt.Parse(nil)
	err := config.ReadConfigFile(CONF_FILE)

	if *showlist {
		showListCommand()
	}

	isScp := false

	if len(*src)==0 && len(*dest)>0 {
			fmt.Println("--src should be setted")
			os.Exit(1)
	}
	if len(*src)>0 && len(*dest)==0 {
			fmt.Println("--dest should be setted")
			os.Exit(1)
	}

	if len(*src)>0 && len(*dest)>0 {
		isScp = true
		if len(*command)>0{
			fmt.Println("The command flag will be ignored")
		}
		if len(*execute)>0{
			fmt.Println("The execute flag will be ignored")
		}
	}

	if len(*pwd) == 0  {
		*pwd, _ = getpass.GetPass()
	}
	hosts, err := config.GetList("groups:" + *group)
	if err != nil {
		fmt.Printf("Group does not exists: %s\n", *group)
		os.Exit(1)
	}

	sshResultChan := make(chan string)
	cmd := *execute
	for _, host := range hosts {
		if isScp{
			// Do some scp stuff
			go executeScp(sshResultChan, host, *src, *dest)
		}else{
			// Execute ssh command
			if len(*command) > 0 {
				cmd, err = config.GetString("commands:" + *command + ":cmd")
				if err != nil {
					fmt.Printf("Command does not exists: %s\n", *command)
					os.Exit(1)
				}
			}
			go executeSsh(sshResultChan, host, cmd)
		}
	}

	fmt.Println("$", cmd, "\n")
	for _, _ = range hosts {
		//Catch the result
		res := <-sshResultChan
		fmt.Println(res)
	}

	if err != nil {
		panic("Failed to dial: " + err.Error())
	}

}
