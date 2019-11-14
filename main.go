package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hornbill/color"
	_ "github.com/hornbill/go-mssqldb" //Microsoft SQL Server driver - v2005+
	_ "github.com/hornbill/mysql"      //MySQL v4.1 to v5.x and MariaDB driver
	_ "github.com/hornbill/mysql320"   //MySQL v3.2.0 to v5 driver - Provides SWSQL (MySQL 4.0.16) support - originally weave-lab
	"github.com/hornbill/sqlx"
)

// main package
func main() {
	//-- Start Time for Durration
	startTime = time.Now()
	//-- Start Time for Log File
	timeNow = time.Now().Format(time.RFC3339)
	timeNow = strings.Replace(timeNow, ":", "-", -1)

	parseFlags()
	//-- Used for Building
	if configVersion {
		fmt.Printf("%v \n", version)
		return
	}
	//-- Output to CLI and Log
	logger(1, "---- Supportworks Call Import Utility V"+fmt.Sprintf("%v", version)+" ----", true)
	logger(1, "Flag - Config File "+configFileName, true)
	logger(1, "Flag - Dry Run "+fmt.Sprintf("%v", configDryRun), true)
	logger(1, "Flag - Concurrent Requests "+fmt.Sprintf("%v", configMaxRoutines), true)

	//Check maxGoroutines for valid value
	maxRoutines, err := strconv.Atoi(configMaxRoutines)
	if err != nil {
		color.Red("Unable to convert maximum concurrency of [" + configMaxRoutines + "] to type INT for processing")
		return
	}
	maxGoroutines = maxRoutines

	if maxGoroutines < 1 || maxGoroutines > 10 {
		color.Red("The maximum concurrent requests allowed is between 1 and 10 (inclusive).\n\n")
		color.Red("You have selected " + configMaxRoutines + ". Please try again, with a valid value against ")
		color.Red("the -concurrent switch.")
		return
	}

	//-- Load Configuration File Into Struct
	swImportConf, boolConfLoaded = loadConfig()
	if boolConfLoaded != true {
		logger(4, "Unable to load config, process closing.", true)
		return
	}

	//Set SQL driver ID string for Application Data
	if swImportConf.SWAppDBConf.Driver == "" {
		logger(4, "SWAppDBConf SQL Driver not set in configuration.", true)
		return
	}
	if swImportConf.SWAppDBConf.Driver == "swsql" {
		appDBDriver = "mysql320"
	} else if swImportConf.SWAppDBConf.Driver == "mysql" || swImportConf.SWAppDBConf.Driver == "mssql" || swImportConf.SWAppDBConf.Driver == "mysql320" {
		appDBDriver = swImportConf.SWAppDBConf.Driver
	} else {
		logger(4, "The SQL driver ("+swImportConf.SWAppDBConf.Driver+") for the Supportworks Application Database specified in the configuration file is not valid.", true)
		return
	}
	//Set SQL driver ID string for Cache Data
	if swImportConf.SWSystemDBConf.Driver == "" {
		logger(4, "SWSystemDBConf SQL Driver not set in configuration.", true)
		return
	}
	if swImportConf.SWSystemDBConf.Driver == "swsql" {
		cacheDBDriver = "mysql320"
	} else if swImportConf.SWSystemDBConf.Driver == "mysql" || swImportConf.SWSystemDBConf.Driver == "mysql320" {
		cacheDBDriver = swImportConf.SWSystemDBConf.Driver
	} else {
		logger(4, "The SQL driver ("+swImportConf.SWSystemDBConf.Driver+") for the Supportworks System Database specified in the configuration file is not valid.", true)
		return
	}

	//-- Log in to Hornbill instance
	var boolLogin = login()
	if boolLogin != true {
		return
	}
	//-- Defer log out of Hornbill instance until after main() is complete
	defer logout()

	//-- Build DB connection strings for sw_systemdb and swdata
	connStrSysDB = buildConnectionString("cache")
	connStrAppDB = buildConnectionString("app")

	var dberr error
	dbsys, dberr = sqlx.Open(cacheDBDriver, connStrSysDB)
	if dberr != nil {
		logger(4, "Could not open cache DB connection"+dberr.Error(), true)
		return
	}
	defer dbsys.Close()

	var db2err error
	dbapp, db2err = sqlx.Open(cacheDBDriver, connStrAppDB)
	if db2err != nil {
		logger(4, "Could not open app DB connection"+db2err.Error(), true)
		return
	}
	defer dbapp.Close()

	loadOrgs()

	//Get request type import config, process each in turn
	for _, val := range swImportConf.RequestTypesToImport {
		if val.Import == true {
			reqPrefix = getRequestPrefix(val.CallClass)
			mapGenericConf = val
			processCallData()
		}
	}

	if len(arrCallsLogged) > 0 {
		//Process associations
		processCallAssociations()
		//Add file attachments to requests
		processAttachments()
	}

	//-- End output
	logger(1, "Requests Logged: "+fmt.Sprintf("%d", counters.created), true)
	logger(1, "Requests Skipped: "+fmt.Sprintf("%d", counters.createdSkipped), true)
	logger(1, "Files Attached: "+fmt.Sprintf("%d", counters.filesAttached), true)
	//-- Show Time Takens
	endTime = time.Now().Sub(startTime)
	logger(1, "Time Taken: "+fmt.Sprintf("%v", endTime), true)
	logger(1, "---- Supportworks Call Import Complete ---- ", true)

}
