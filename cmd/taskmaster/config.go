package main

import (
	"io/ioutil"
	"os"
	"sort"
	"syscall"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Name         string // name of program
	Sig          os.Signal
	Cmd          string            `yaml:"cmd"`      // binary to run
	Args         []string          `yaml:"args"`     // list of args
	NumProcs     int               `yaml:"numprocs"` // number of processes
	Umask        int               `yaml:"umask"`    // int representing permissions
	WorkingDir   string            `yaml:"workingdir"`
	AutoStart    bool              `yaml:"autostart"`    // true/false (default: false)
	AutoRestart  string            `yaml:"autorestart"`  // always/never/unexpected (defult: never)
	ExitCodes    []int             `yaml:"exitcodes"`    // expected exit codes (default: 0)
	StartRetries int               `yaml:"startretries"` // times to retry if unexpected exit
	StartTime    int               `yaml:"starttime"`    // delay before start
	StopSignal   string            `yaml:"stopsignal"`   // if time up what signal to send
	StopTime     int               `yaml:"stoptime"`     // time until signal sent
	Stdin        string            `yaml:"stdin"`        // file read as stdin
	Stdout       string            `yaml:"stdout"`       // stdout redirect file
	Stderr       string            `yaml:"stderr"`       // stderr redirect file
	Env          map[string]string `yaml:"env"`          // map of env vars
}

func ParseConfig(filename string) (map[string]Config, error) {
	ymap := make(map[interface{}]interface{})

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal([]byte(data), &ymap)
	if err != nil {
		return nil, err
	}

	confs := make(map[string]Config)
	for k, v := range ymap["programs"].(map[interface{}]interface{}) {
		conf := Config{}
		data, err := yaml.Marshal(v)
		if err != nil {
			return confs, err
		}
		err = yaml.Unmarshal(data, &conf)
		if err != nil {
			return confs, err
		}

		// set defaults
		conf.Sig = syscall.SIGINT // TODO: set signal properly
		if len(conf.ExitCodes) == 0 {
			conf.ExitCodes = []int{0}
		}
		sort.Ints(conf.ExitCodes)
		if conf.AutoRestart == "" {
			conf.AutoRestart = "unexpected"
		}
		conf.Name = k.(string)
		confs[conf.Name] = conf
	}
	return confs, nil
}

func UpdateConfig(file string, old ProcessMap, p ProcChans) ProcessMap {
	new, err := ParseConfig(file) //Make it return ProcessMap?
	if err != nil {
		logger.Println(err)
		panic(err) //Panic? or print erro and keep running same? or catch panic outside
	}
	tmp := ConfigToProcess(new)
	for i, slices := range tmp {
		_, ok := old[i]
		if !ok {
			for _, v := range slices {
				p.newPros <- v //Addeding
			}
		} else { //already running
			tmp[i] = old[i]
			//need to check if it's been changed or not and restarted?
			delete(old, i)
		}
	}
	for _, slices := range old {
		for _, v := range slices {
			p.oldPros <- v //removing
		}
	}
	return tmp
}
