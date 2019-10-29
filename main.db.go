package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/hornbill/sqlx"
)

//buildConnectionString -- Build the connection string for the SQL driver
func buildConnectionString(strDataSource string) string {
	connectString := ""
	if strDataSource == "app" {
		//Build
		if appDBDriver == "" || swImportConf.SWAppDBConf.Server == "" || swImportConf.SWAppDBConf.Database == "" || swImportConf.SWAppDBConf.UserName == "" || swImportConf.SWAppDBConf.Port == 0 {
			logger(4, "Application Database configuration not set.", true)
			return ""
		}
		switch appDBDriver {
		case "mssql":
			connectString = "server=" + swImportConf.SWAppDBConf.Server
			connectString = connectString + ";database=" + swImportConf.SWAppDBConf.Database
			connectString = connectString + ";user id=" + swImportConf.SWAppDBConf.UserName
			connectString = connectString + ";password=" + swImportConf.SWAppDBConf.Password
			if swImportConf.SWAppDBConf.Encrypt == false {
				connectString = connectString + ";encrypt=disable"
			}
			if swImportConf.SWAppDBConf.Port != 0 {
				dbPortSetting := strconv.Itoa(swImportConf.SWAppDBConf.Port)
				connectString = connectString + ";port=" + dbPortSetting
			}
		case "mysql":
			connectString = swImportConf.SWAppDBConf.UserName + ":" + swImportConf.SWAppDBConf.Password
			connectString = connectString + "@tcp(" + swImportConf.SWAppDBConf.Server + ":"
			if swImportConf.SWAppDBConf.Port != 0 {
				dbPortSetting := strconv.Itoa(swImportConf.SWAppDBConf.Port)
				connectString = connectString + dbPortSetting
			} else {
				connectString = connectString + "3306"
			}
			connectString = connectString + ")/" + swImportConf.SWAppDBConf.Database

		case "mysql320":
			dbPortSetting := strconv.Itoa(swImportConf.SWAppDBConf.Port)
			connectString = "tcp:" + swImportConf.SWAppDBConf.Server + ":" + dbPortSetting
			connectString = connectString + "*" + swImportConf.SWAppDBConf.Database + "/" + swImportConf.SWAppDBConf.UserName + "/" + swImportConf.SWAppDBConf.Password
		}
	} else if strDataSource == "cache" {
		//Build & return sw_systemdb connection string
		if swImportConf.SWServerAddress == "" {
			logger(4, "Supportworks Server Address not set.", true)
			return ""
		}
		if swImportConf.SWSystemDBConf.UserName == "" {
			logger(4, "System Database configuration not set.", true)
			return ""
		}
		connectString = "tcp:" + swImportConf.SWServerAddress + ":5002"
		connectString = connectString + "*sw_systemdb/" + swImportConf.SWSystemDBConf.UserName + "/" + swImportConf.SWSystemDBConf.Password

	}
	return connectString
}

//queryDBCallDetails -- Query call data & set map of calls to add to Hornbill
func queryDBCallDetails(callClass, swCallClass, connString string) bool {
	if callClass == "" || connString == "" {
		return false
	}

	//Connect to the JSON specified DB
	db2, err := sqlx.Open(appDBDriver, connString)
	if err != nil {
		logger(4, "[DATABASE] Database Connection Error: "+fmt.Sprintf("%v", err), true)
		return false
	}
	defer db2.Close()
	//Check connection is open
	err = db2.Ping()
	if err != nil {
		logger(4, "[DATABASE] [PING] Database Connection Error: "+fmt.Sprintf("%v", err), true)
		return false
	}
	logger(3, "[DATABASE] Connection Successful", true)
	logger(3, "[DATABASE] Retrieving "+callClass+"s, "+swCallClass+" from Supportworks.", true)
	logger(3, "[DATABASE] Please Wait...", true)
	//build query
	sqlCallQuery = mapGenericConf.SQLStatement
	logger(3, "[DATABASE] Query to retrieve "+callClass+" calls from Supportworks: "+sqlCallQuery, false)

	//Run Query
	rows, err := db2.Queryx(sqlCallQuery)
	if err != nil {
		logger(4, " Database Query Error: "+fmt.Sprintf("%v", err), true)
		return false
	}
	defer rows.Close()
	//Clear down existing Call Details map
	arrCallDetailsMaps = nil
	//Build map full of calls to import
	intCallCount := 0
	for rows.Next() {
		intCallCount++
		results := make(map[string]interface{})
		err = rows.MapScan(results)
		if err != nil {
			//something is wrong with this row just log then skip it
			logger(4, " Database Result error"+err.Error(), true)
			continue
		}
		//Stick marshalled data map in to parent slice
		arrCallDetailsMaps = append(arrCallDetailsMaps, results)
	}
	return true
}

// getFieldValue --Retrieve field value from mapping via SQL record map
func getFieldValue(v string, u map[string]interface{}) string {
	fieldMap := v
	//-- Match $variable from String
	re1, err := regexp.Compile(`\[(.*?)\]`)
	if err != nil {
		color.Red("[ERROR] %v", err)
	}

	result := re1.FindAllString(fieldMap, 100)
	valFieldMap := ""
	//-- Loop Matches
	for _, val := range result {
		valFieldMap = ""
		valFieldMap = strings.Replace(val, "[", "", 1)
		valFieldMap = strings.Replace(valFieldMap, "]", "", 1)
		if valFieldMap == "oldCallRef" {
			valFieldMap = "h_formattedcallref"
			if u[valFieldMap] != nil {

				if valField, ok := u[valFieldMap].(int64); ok {
					valFieldMap = strconv.FormatInt(valField, 10)
				} else {
					valFieldMap = fmt.Sprintf("%+s", u[valFieldMap])
				}

				if valFieldMap != "<nil>" {
					fieldMap = strings.Replace(fieldMap, val, valFieldMap, 1)
				}

			} else {
				valFieldMap = "callref"
				if u[valFieldMap] != nil {

					if valField, ok := u[valFieldMap].(int64); ok {
						valFieldMap = strconv.FormatInt(valField, 10)
					} else {
						valFieldMap = fmt.Sprintf("%+s", u[valFieldMap])
					}

					if valFieldMap != "<nil>" {
						fieldMap = strings.Replace(fieldMap, val, padCallRef(valFieldMap, "F", 7), 1)
					}
				} else {
					fieldMap = strings.Replace(fieldMap, val, "", 1)
				}
			}
		} else {
			if u[valFieldMap] != nil {

				if valField, ok := u[valFieldMap].(int64); ok {
					valFieldMap = strconv.FormatInt(valField, 10)
				} else {
					valFieldMap = fmt.Sprintf("%+s", u[valFieldMap])
				}

				if valFieldMap != "<nil>" {
					fieldMap = strings.Replace(fieldMap, val, valFieldMap, 1)
				}
			} else {
				fieldMap = strings.Replace(fieldMap, val, "", 1)
			}
		}
	}
	return fieldMap
}
