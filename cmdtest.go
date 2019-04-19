package main

import (
	"fmt"
	"log"
	"os/exec"
)

func main() {
	cmd := "ip a | grep ens33|grep inet|awk '{print $2}'"
	c := exec.Command("bash", "-c", cmd)
	out, err := c.CombinedOutput()
	if err != nil {
		log.Println(err)
	}

	fmt.Println(string(out))

}
