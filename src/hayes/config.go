package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type ConfigType struct {            
	registers Registers
	
	// Configuration
	echoInCmdMode bool       
	speakerMode int          
	speakerVolume int        
	verbose bool             
	quiet bool               
	connectMsgSpeed bool     
	busyDetect bool          
	extendedResultCodes bool 
	dcdControl bool          
}
type storedConfigs struct {
	powerUpConfig int        
	Config [3]ConfigType
	currentConfig int
}
var profile storedConfigs
var config ConfigType

type jsonConfig struct {
	PowerUpConfig int                `json:"PowerUpConfig"`
	Config [2]struct {              // `json:"Config"`
		Regs map[string]byte     `json:"Regs"`
		
		// Configuration
		EchoInCmdMode bool       `json:"EchoInCmdMode"`
		SpeakerMode int          `json:"SpeakerMode"`
		SpeakerVolume int        `json:"SpeakerVolume"`
		Verbose bool             `json:"Verbose"`
		Quiet bool               `json:"Quiet"`
		ConnectMsgSpeed bool     `json:"ConnectMsgSpeed"`
		BusyDetect bool          `json:"BusyDetect"`
		ExtendedResultCodes bool `json:"ExtendedResultCodes"`
		DCDControl bool          `json:"DCDControl"`
	}
}

func resetConfig() ConfigType {
	var m ConfigType
	m.echoInCmdMode = true  // Echo local keypresses
	m.quiet = false		// Modem offers return status
	m.verbose = true	// Text return codes
	m.speakerVolume = 1	// moderate volume
	m.speakerMode = 1	// on until other modem heard
	m.busyDetect = true
	m.extendedResultCodes = true
	m.dcdControl = false	
	m.connectMsgSpeed = true

	return m
}

// Need:
// - Need function to swap current config into one of the stored profiles
// - Need function to swap one of the stored profiles into current config
// - Abstract StoredConfig (call it Profiles), w/ methods as needed.
// 

func profileString(m ConfigType) string {
	b := func(p bool) (string) {
		if p {
			return"1 "
		} 
		return "0 "
	};
	i := func(p int) (string) {
		return fmt.Sprintf("%d ", p)
	};
	x := func(r, b bool) (string) {
		if (r == false && b == false) {
			return "0 "
		}
		if (r == true && b == false) {
			return "1 "
		}
		if (r == true && b == true) {
			return "7 "
		}
		return "0 "
	};

	s := "E" + b(m.echoInCmdMode)
	s += "F1 "		// For Hayes 1200 compatability 
	s += "L" + i(m.speakerVolume)
	s += "M" + i(m.speakerMode)
	s += "Q" + b(m.quiet)
	s += "V" + b(m.verbose)
	s += "W" + b(m.connectMsgSpeed)
	s += "X" + x(m.extendedResultCodes, m.busyDetect)
	s += "&C" + b(m.dcdControl)
	s += "\n"
	s += m.registers.String()

	return s
}

func (c *storedConfigs) String() string {
	return profileString(c.Config[0]) + profileString(c.Config[1])
}

func (c *storedConfigs) Active() *ConfigType { return &c.Config[c.currentConfig] }

// TODO: this needs to make a copy of i, not return it.
//       and need to add 1 to i
func (c *storedConfigs) Switch(i int) *ConfigType {
	return &c.Config[i]
}


// TODO: DEFAULTS?
func (c *storedConfigs) loadStoredConfigs() error {
	var newconf storedConfigs
	var j jsonConfig
	
	b, err := ioutil.ReadFile("hayes.config.json")
	if err != nil {
		e := fmt.Errorf("Can't read config file: %s", err)
		logger.Print(e)
		return e
	}

	if err = json.Unmarshal(b, &j); err != nil {
		logger.Print(err)
		return err
	}

	newconf.powerUpConfig = j.PowerUpConfig
	for i :=0; i < 3; i++ {
		newconf.Config[i].registers =
			registersJsonUnmap(j.Config[i].Regs)
		newconf.Config[i].echoInCmdMode = j.Config[i].EchoInCmdMode
		newconf.Config[i].speakerVolume =j.Config[i].SpeakerVolume
		newconf.Config[i].speakerMode = j.Config[i].SpeakerMode
		newconf.Config[i].quiet = j.Config[i].Quiet
		newconf.Config[i].verbose = j.Config[i].Verbose
		newconf.Config[i].connectMsgSpeed = j.Config[i].ConnectMsgSpeed
		newconf.Config[i].extendedResultCodes =
			j.Config[i].ExtendedResultCodes
		newconf.Config[i].busyDetect = j.Config[i].BusyDetect
		newconf.Config[i].dcdControl = j.Config[i].DCDControl
	}

	c = &newconf
	return nil
}

// AT&Wn
// TODO: THis needs to be done.

func (c *storedConfigs) writeActiveConfig(i int) error { return nil }

func (c *storedConfigs) writeConfig() error {
	var j jsonConfig

	j.PowerUpConfig = c.powerUpConfig
	for i :=0; i < 3; i++ {
		j.Config[i].Regs = c.Config[i].registers.JsonMap()
		j.Config[i].EchoInCmdMode = c.Config[i].echoInCmdMode
		j.Config[i].SpeakerVolume = c.Config[i].speakerVolume
		j.Config[i].SpeakerMode = c.Config[i].speakerMode
		j.Config[i].Quiet = c.Config[i].quiet
		j.Config[i].Verbose = c.Config[i].verbose
		j.Config[i].ConnectMsgSpeed = c.Config[i].connectMsgSpeed
		j.Config[i].ExtendedResultCodes = c.Config[i].extendedResultCodes
		j.Config[i].BusyDetect = c.Config[i].busyDetect
		j.Config[i].DCDControl = c.Config[i].dcdControl
	}
	

	b, err := json.MarshalIndent(j, "", "\t")
	if err != nil {
		logger.Print(err)
		return err
	}
	err = ioutil.WriteFile("hayes.config.json", b, 0644)
	if err != nil {
		logger.Print(err)
	}
	return err
}

func bootstrap() {
	var c storedConfigs
	c.powerUpConfig = 0
	c.Config[0] = resetConfig()
	c.Config[0].registers = NewRegisters()
	c.Config[0].registers.Reset()
	c.Config[1] = resetConfig()
	c.Config[1].registers = NewRegisters()
	c.Config[1].registers.Reset()

	c.writeConfig()
}	

// TODO: defaults?
func (c *storedConfigs) switchStoredConfig(confnum int) error {
/* 
	if confnum != 0 && confnum != 1 {
		err := fmt.Errorf("Invalid config number %d", confnum)
		logger.Print(err)
		return err
	}
	logger.Printf("Switching to stored config %d", confnum)

	registers = registers.JsonUnmap(config.Config[confnum].Regs)

	commands, err := parseCommand(config.Config[confnum].InitCmd)
	if err != nil {
		logger.Print(err)
		return err
	}

	for _, cmd := range commands {
		err = processSingleCommand(cmd)
		if err != nil {
			break
		}
	}

	if err != nil {
		factoryReset()
	}
*/
	return nil
}


// AT&Y
func (c *storedConfigs) setPowerUpConfig(i int) error {
	if i != 0 && i != 1 {
		return fmt.Errorf("Invalid config number %d", i)
	}
	c.powerUpConfig = i
	c.writeConfig()
	return OK
}

// ATZn - 0 == config 0, 1 == config 1
func softReset(i int) error {
	factoryReset()
	return profile.switchStoredConfig(i)
}

// AT&F - reset to factory defaults
func factoryReset() error {
	var err error = OK

	logger.Print("Resetting modem")

	// Reset state
	goOnHook()
	setLineBusy(false)
	lowerDSR()
	lowerCTS()
	lowerRI()
	stopTimer()
	m.dcd = false
	m.lastCmd = ""
	m.lastDialed = ""
	m.connectSpeed = 0

	registers.Reset()
	phonebook = NewPhonebook(*_flags_phoneBook, logger)
	err = phonebook.Load()
	if err != nil {
		logger.Print(err)
	}

	config = resetConfig()
	
	resetTimer()
	return err
}

