package pilot

import (
	"time"
	"fmt"
	"os/exec"
	"os"
	"strconv"

	log "github.com/Sirupsen/logrus"
)

const (
	ENV_CLEANER_LOG_ROTATE = "PILOT_CLEANER_LOG_ROTATE"
)

type CleanJob struct {
	Name         string
	ContainerDir string
	HostDir      string
	File         string
}

type ContainerConfig struct {
	ID   string
	Name string
	Jobs []*CleanJob
}

type Cleaner struct {
	Configs    map[string]*ContainerConfig
	UpdateChan chan *ContainerConfig
	DeleteChan chan string
	LogRotate  int
}

func NewCleaner() *Cleaner {
	cleaner := &Cleaner{}
	cleaner.Configs = make(map[string]*ContainerConfig)
	cleaner.UpdateChan = make(chan *ContainerConfig)
	cleaner.DeleteChan = make(chan string)
	cleaner.LogRotate = 7;

	logRoate := os.Getenv(ENV_CLEANER_LOG_ROTATE)
	if logRoate != "" {
		rotate, err := strconv.Atoi(logRoate)
		if err == nil {
			cleaner.LogRotate = rotate
		}
	}

	return cleaner
}

func (c *Cleaner) Run() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case deleteID := <-c.DeleteChan:
			delete(c.Configs, deleteID)
			log.Infof("cleaner delete config %s", deleteID)
		case updateConfig := <-c.UpdateChan:
			c.Configs[updateConfig.ID] = updateConfig
			log.Infof("cleaner update config %s %v", updateConfig.ID, updateConfig)
		case <-ticker.C:
			log.Info("cleaner start to clean ...")
			for _, config := range c.Configs {
				log.Infof("cleaner container %s", config.Name)
				for _, job := range config.Jobs {
					log.Infof("cleaner remove log %s,%s", job.Name, job.ContainerDir)
					output, err := job.RemoveLog(c.LogRotate)
					if err != nil {
						log.Errorf("cleaner remove error %s", err)
					}
					log.Debug("cleaner remove output \n", output)
				}
			}
			log.Info("cleaner finish to clean.")
		}
	}
}

func (c *Cleaner) DeleteConfig(id string) {
	c.DeleteChan <- id
}

func (c *Cleaner) UpdateConfig(containerId string, container map[string]string, configList []*LogConfig) {
	containerConfig := &ContainerConfig{}

	containerConfig.ID = containerId
	if value, ok := container["docker_container"]; ok {
		containerConfig.Name = value
	}

	containerConfig.Jobs = make([]*CleanJob, 0)
	for _, log := range configList {
		//stdout的日志不用清理
		if log.Stdout {
			continue
		}

		job := &CleanJob{}
		job.Name = log.Name
		job.ContainerDir = log.ContainerDir
		job.HostDir = log.HostDir
		job.File = log.File
		containerConfig.Jobs = append(containerConfig.Jobs, job)
	}

	c.UpdateChan <- containerConfig
}

func (j *CleanJob) RemoveLog(logRotate int) (string, error) {
	//有的日志切分后用日期做后缀,查询日志的正则在后面加一个*
	//cmd := fmt.Sprintf("ionice -c 2 -n 7 find %s -name \"%s*\" -type f -mtime +%d -exec rm -f {} \\;", j.HostDir, j.File, logRotate)
	cmd := fmt.Sprintf("ionice -c 2 -n 7 find %s -name \"%s*\" -not -name \"error.log\" -type f -mtime +%d -exec rm -f {} \\;", j.HostDir, j.File, logRotate)
	log.Debugf("remove method, cmd: %s", cmd)

	cls := exec.Command("/bin/sh", "-c", cmd)

	output, err := cls.CombinedOutput()
	return string(output), err
}
