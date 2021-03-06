package rvn

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	log "github.com/sirupsen/logrus"
)

func GenConfigAll(topo Topo) {
	for _, node := range topo.Nodes {
		GenConfig(node.Host, topo)
	}
	for _, zwitch := range topo.Switches {
		GenConfig(zwitch.Host, topo)
	}
}

func GenConfig(h Host, topo Topo) {
	tp_path, err := filepath.Abs("/var/rvn/template/config.yml")
	if err != nil {
		log.Printf("failed to create absolute path for config.yml - %v", err)
		return
	}
	tp, err := template.ParseFiles(tp_path)
	if err != nil {
		log.Printf("failed to read config.yml - %v", err)
		return
	}

	wd, err := WkDir()
	if err != nil {
		log.Printf("genconfig: failed to get working dir")
		return
	}

	path := fmt.Sprintf("/%s/%s.yml", wd, h.Name)
	f, err := os.Create(path)
	if err != nil {
		log.Printf("failed to create path %s - %v", path, err)
		return
	}
	defer f.Close()

	data := struct {
		Host Host
		NFS  string
	}{h, topo.MgmtIp}

	err = tp.Execute(f, data)
	if err != nil {
		log.Printf("failed to execute config template for %s - %v", h.Name, err)
	}
}

func Configure(withUserConfig bool) {
	//TODO return error condition

	topo, err := LoadTopo()
	if err != nil {
		if strings.Contains(err.Error(), "topo.json: no such file or directory") {
			log.Printf("Topology not built. Use `rvn build` first")
			return
		}
		log.Println("configure: failed to load topo - %v", err)
		return
	}
	preConfigure(topo)
	status := Status()
	node_status := status["nodes"].(map[string]DomStatus)
	switch_status := status["switches"].(map[string]DomStatus)

	var wg sync.WaitGroup
	doConfig := func(topo Topo, host Host, ds DomStatus) {

		configureNode(topo, host, ds, withUserConfig)
		wg.Done()
	}

	for _, x := range topo.Nodes {
		if x.OS == "netboot" {
			continue
		}
		wg.Add(1)
		go doConfig(topo, x.Host, node_status[x.Name])
	}
	for _, x := range topo.Switches {
		wg.Add(1)
		go doConfig(topo, x.Host, switch_status[x.Name])
	}

	wg.Wait()

	log.Println("configuration of all nodes complete")
}

func preConfigure(topo Topo) {
	pc_script := fmt.Sprintf("%s/pre-config/run", topo.Dir)
	if _, err := os.Stat(pc_script); err == nil {
		log.Printf("running pre-config for %s", topo.Name)

		wd, err := WkDir()
		if err != nil {
			log.Printf("preconfigure: failed to get working dir")
			return
		}

		cmd := exec.Command(pc_script)
		env := os.Environ()
		cmd.Env = append(env, fmt.Sprintf("TOPOJSON=%s/topo.json", wd))
		cmd.Dir = fmt.Sprintf("%s/pre-config", topo.Dir)
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("pre-config failed %s - %v", out, err)
		}
	}

}

func ConfigureNode(topo Topo, node string) {
	status := Status()
	node_status := status["nodes"].(map[string]DomStatus)
	switch_status := status["switches"].(map[string]DomStatus)

	s, ok := node_status[node]
	h := topo.getHost(node)
	if ok {
		configureNode(topo, *h, s, true)
	} else if s, ok := switch_status[node]; ok {
		configureNode(topo, *h, s, true)
	}
}

func ConfigureNodes(topo Topo, nodes []string) {
	status := Status()
	node_status := status["nodes"].(map[string]DomStatus)
	switch_status := status["switches"].(map[string]DomStatus)

	var wg sync.WaitGroup
	doConfig := func(topo Topo, host string, ds DomStatus) {

		s, ok := node_status[host]
		h := topo.getHost(host)
		if ok {
			configureNode(topo, *h, s, true)
		} else if s, ok := switch_status[host]; ok {
			configureNode(topo, *h, s, true)
		}

		wg.Done()

	}

	for _, x := range nodes {
		wg.Add(1)
		go doConfig(topo, x, node_status[x])
	}

	wg.Wait()

	log.Println("configuration of all nodes complete")

}

func configureNode(topo Topo, host Host, ds DomStatus, withUserConfig bool) {

	wd, err := WkDir()
	if err != nil {
		log.Printf("configure: failed to get working dir")
		return
	}

	yml := fmt.Sprintf("%s/%s.yml", wd, host.Name)
	log.Printf("running base config for %s:%s", topo.Name, host.Name)
	runAnsible(yml, topo.Name, host, ds)

	user_yml := fmt.Sprintf("%s/config/%s.yml", topo.Dir, host.Name)
	if _, err := os.Stat(user_yml); err == nil {
		if withUserConfig {
			log.Printf("running user config for %s:%s", topo.Name, host.Name)
			runAnsible(user_yml, topo.Name, host, ds)
		}
	}

}

func runAnsible(yml, topo string, h Host, s DomStatus) {

	dbCheckConnection()
	db_state_key := fmt.Sprintf("config_state:%s:%s", topo, h.Name)
	db.Set(db_state_key, "configuring", 0)

	if strings.ToLower(h.OS) == "netboot" {
		return
	}

	extra_vars := "ansible_become_pass=rvn"
	if strings.ToLower(h.OS) == "freebsd" {
		extra_vars += " ansible_python_interpreter='/usr/local/bin/python2'"
	}

	cmd := exec.Command(
		"ansible-playbook",
		"-i", s.IP+",",
		yml,
		"--extra-vars", extra_vars,
		`--ssh-extra-args='-i/var/rvn/ssh/rvn'`,
		"--user=rvn", "--private-key=/var/rvn/ssh/rvn",
	)
	cmd.Env = append(os.Environ(), "ANSIBLE_HOST_KEY_CHECKING=False")
	out, err := cmd.CombinedOutput()

	if err != nil {
		log.Printf("failed to run configuration for %s - %v", h.Name, err)
		log.Printf(string(out))
		db.Set(db_state_key, "failed", 0)
	} else {
		db.Set(db_state_key, "success", 0)
	}

}
