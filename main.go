package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/tcnksm/go-latest" //-- For Version checking

	"github.com/hornbill/sqlx"
	//SQL Drivers
	_ "github.com/alexbrainman/odbc"
	_ "github.com/hornbill/go-mssqldb" //Microsoft SQL Server driver - v2005+
	_ "github.com/hornbill/mysql"      //MySQL v4.1 to v5.x and MariaDB driver
	_ "github.com/hornbill/mysql320"   //MySQL v3.2.0 to v5 driver - Provides SWSQL (MySQL 4.0.16) support - originally weave-lab
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
	checkVersion()
	logger(1, "Flag - Config File "+configFileName, true)
	logger(1, "Flag - Dry Run "+fmt.Sprintf("%v", configDryRun), true)
	logger(1, "Flag - Concurrent Requests "+fmt.Sprintf("%v", configMaxRoutines), true)

	//Check maxGoroutines for valid value
	maxRoutines, err := strconv.Atoi(configMaxRoutines)
	if err != nil {
		logger(4, "Unable to convert maximum concurrency of ["+configMaxRoutines+"] to type INT for processing", true)
		return
	}
	maxGoroutines = maxRoutines

	if maxGoroutines < 1 || maxGoroutines > 10 {
		logger(4, "The maximum concurrent requests allowed is between 1 and 10 (inclusive).\n\n", true)
		logger(4, "You have selected "+configMaxRoutines+". Please try again, with a valid value against ", true)
		logger(4, "the -concurrent switch.", true)
		return
	}

	//-- Load Configuration File Into Struct
	swImportConf, boolConfLoaded = loadConfig()
	if !boolConfLoaded {
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
	} else if swImportConf.SWAppDBConf.Driver == "mysql" || swImportConf.SWAppDBConf.Driver == "mssql" || swImportConf.SWAppDBConf.Driver == "mysql320" || swImportConf.SWAppDBConf.Driver == "odbc" || swImportConf.SWAppDBConf.Driver == "ODBC" {
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

	if swImportConf.HBConf.APIKey == "" {
		//-- Log in to Hornbill instance
		var boolLogin = login()
		if !boolLogin {
			return
		}
		//-- Defer log out of Hornbill instance until after main() is complete
		defer logout()
	}

	//-- Build DB connection strings for sw_systemdb and swdata
	connStrSysDB = buildConnectionString("cache")
	connStrAppDB = buildConnectionString("app")

	var db2err error
	//fmt.Println(connStrAppDB)
	dbapp, db2err = sqlx.Open(appDBDriver, connStrAppDB)
	if db2err != nil {
		logger(4, "Could not open app DB connection"+db2err.Error(), true)
		return
	}
	defer dbapp.Close()

	if swImportConf.SWSystemDBConf.Driver == "mysql" && swImportConf.SWSystemDBConf.Driver == swImportConf.SWAppDBConf.Driver {
		dbsys = dbapp
	} else {
		var dberr error
		dbsys, dberr = sqlx.Open(cacheDBDriver, connStrSysDB)
		if dberr != nil {
			logger(4, "Could not open cache DB connection"+dberr.Error(), true)
			return
		}
		defer dbsys.Close()
	}

	err = loadOrgs()
	if err != nil {
		logger(4, "Error when trying to cache Organisation records from instance: "+err.Error(), true)
	}

	//Get request type import config, process each in turn
	for _, val := range swImportConf.RequestTypesToImport {
		if val.Import {
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
	logger(1, "Requests Returned: "+fmt.Sprintf("%d", counters.callsReturned), true)
	logger(1, "Requests Logged: "+fmt.Sprintf("%d", counters.created), true)
	logger(1, "Requests Skipped: "+fmt.Sprintf("%d", counters.createdSkipped), true)
	if counters.existingRequests > 0 {
		logger(1, "Existing Requests Processed: "+fmt.Sprintf("%d", counters.existingRequests), true)
	}
	logger(1, "Files Attached: "+fmt.Sprintf("%d", counters.filesAttached), true)
	//-- Show Time Takens
	endTime = time.Since(startTime)
	logger(1, "Time Taken: "+fmt.Sprintf("%v", endTime), true)
	logger(1, "---- Supportworks Call Import Complete ---- ", true)

}

//-- Check Latest
func checkVersion() {
	githubTag := &latest.GithubTag{
		Owner:      "hornbill",
		Repository: repo,
	}

	res, err := latest.Check(githubTag, version)
	if err != nil {
		logger(4, "Unable to check utility version against Github repository: "+err.Error(), true)
		return
	}
	if res.Outdated {
		logger(5, version+" is not latest, you should upgrade to "+res.Current+" by downloading the latest package Here https://github.com/hornbill/"+repo+"/releases/tag/v"+res.Current, true)
	}
}
