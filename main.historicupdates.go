package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	apiLib "github.com/hornbill/goApiLib"
	"github.com/hornbill/sqlx"
	//SQL Drivers
	_ "github.com/alexbrainman/odbc"
)

//applyHistoricalUpdates - takes call diary records from Supportworks, imports to Hornbill as Historical Updates
func applyHistoricalUpdates(request RequestReferences, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) {

	smCallRef := request.SmCallID
	swCallRef := request.SwCallID

	db2, err := sqlx.Open(appDBDriver, connStrAppDB)
	if err != nil {
		logger(4, "[DATABASE] Database Connection Error: "+err.Error(), true)
		return
	}
	defer db2.Close()
	diaryQuery := strings.ReplaceAll(swImportConf.CallDiaryQuery, "[sourceref]", swCallRef)
	if configDebug {
		buffer.WriteString(loggerGen(3, "[DATABASE] Connection Successful"))
		buffer.WriteString(loggerGen(3, "[DATABASE] Running query for Historical Updates of call "+swCallRef+". Please wait..."))
		buffer.WriteString(loggerGen(3, "[DATABASE] Diary Query: "+diaryQuery))
	}

	//Run Query
	rows, err := db2.Queryx(diaryQuery)
	if err != nil {
		buffer.WriteString(loggerGen(4, " Database Query Error: "+err.Error()))
		return
	}
	defer rows.Close()
	sucCount := 0
	errCount := 0
	//Process each call diary entry, insert in to Hornbill
	for rows.Next() {
		diaryEntry := make(map[string]interface{})
		err = rows.MapScan(diaryEntry)
		if err != nil {
			buffer.WriteString(loggerGen(4, "Unable to retrieve data from SQL query: "+err.Error()))
			errCount++
		} else {
			//Update Time - EPOCH to Date/Time Conversion
			diaryTime := ""
			if diaryEntry["updatetimex"] != nil {
				diaryTimex := ""
				if updateTime, ok := diaryEntry["updatetimex"].(int64); ok {
					diaryTimex = strconv.FormatInt(updateTime, 10)
				} else {
					diaryTimex = fmt.Sprintf("%+s", diaryEntry["updatetimex"])
				}
				diaryTime = epochToDateTime(diaryTimex)
			}

			//Check for source/code/text having nil value
			diarySource := ""
			if diaryEntry["udsource"] != nil {
				diarySource = fmt.Sprintf("%+s", diaryEntry["udsource"])
			}

			diaryCode := ""
			if diaryEntry["udcode"] != nil {
				diaryCode = fmt.Sprintf("%+s", diaryEntry["udcode"])
			}

			diaryText := ""
			if diaryEntry["updatetxt"] != nil {
				diaryText = fmt.Sprintf("%+s", diaryEntry["updatetxt"])
			}

			diaryIndex := ""
			if diaryEntry["udindex"] != nil {
				if updateIndex, ok := diaryEntry["udindex"].(int64); ok {
					diaryIndex = strconv.FormatInt(updateIndex, 10)
				} else {
					diaryIndex = fmt.Sprintf("%+s", diaryEntry["udindex"])
				}
			}

			diaryTimeSpent := ""
			if diaryEntry["timespent"] != nil {
				if updateSpent, ok := diaryEntry["timespent"].(int64); ok {
					diaryTimeSpent = strconv.FormatInt(updateSpent, 10)
				} else {
					diaryTimeSpent = fmt.Sprintf("%+s", diaryEntry["timespent"])
				}
			}

			diaryType := ""
			if diaryEntry["udtype"] != nil {
				if updateType, ok := diaryEntry["udtype"].(int64); ok {
					diaryType = strconv.FormatInt(updateType, 10)
				} else {
					diaryType = fmt.Sprintf("%+s", diaryEntry["udtype"])
				}
			}

			espXmlmc.SetParam("application", appServiceManager)
			espXmlmc.SetParam("entity", "RequestHistoricUpdates")
			espXmlmc.OpenElement("primaryEntityData")
			espXmlmc.OpenElement("record")
			espXmlmc.SetParam("h_fk_reference", smCallRef)
			espXmlmc.SetParam("h_updatedate", diaryTime)
			if diaryTimeSpent != "" && diaryTimeSpent != "0" {
				espXmlmc.SetParam("h_timespent", diaryTimeSpent)
			}
			if diaryType != "" {
				espXmlmc.SetParam("h_updatetype", diaryType)
			}
			espXmlmc.SetParam("h_updatebytype", "1")
			espXmlmc.SetParam("h_updateindex", diaryIndex)
			if diaryEntry["repid"] != nil {
				espXmlmc.SetParam("h_updateby", fmt.Sprintf("%+s", diaryEntry["repid"]))
			}
			if diaryEntry["repid"] != nil {
				espXmlmc.SetParam("h_updatebyname", fmt.Sprintf("%+s", diaryEntry["repid"]))
			}
			if diaryEntry["groupid"] != nil {
				espXmlmc.SetParam("h_updatebygroup", fmt.Sprintf("%+s", diaryEntry["groupid"]))
			}
			if diaryCode != "" {
				espXmlmc.SetParam("h_actiontype", diaryCode)
			}
			if diarySource != "" {
				espXmlmc.SetParam("h_actionsource", diarySource)
			}
			if diaryText != "" {
				espXmlmc.SetParam("h_description", diaryText)
			}
			espXmlmc.CloseElement("record")
			espXmlmc.CloseElement("primaryEntityData")

			if configDebug {
				buffer.WriteString(loggerGen(3, "XMLMC data::entityAddRecord::RequestHistoricUpdates: "+espXmlmc.GetParam()))
			}
			XMLUpdate, xmlmcErr := espXmlmc.Invoke("data", "entityAddRecord")
			if xmlmcErr != nil {
				buffer.WriteString(loggerGen(3, "API Invoke Failed Unable to add Historical Call Diary Update: "+xmlmcErr.Error()))
				errCount++
			}
			var xmlRespon xmlmcResponse
			errXMLMC := xml.Unmarshal([]byte(XMLUpdate), &xmlRespon)
			if errXMLMC != nil {
				buffer.WriteString(loggerGen(4, "Unable to read response from Hornbill instance: "+errXMLMC.Error()))
				errCount++
			}
			if xmlRespon.MethodResult != "ok" {
				buffer.WriteString(loggerGen(3, "API Call Failed Unable to add Historical Call Diary Update: "+xmlRespon.State.ErrorRet))
				errCount++
			}
			sucCount++

		}
	}
	buffer.WriteString(loggerGen(1, strconv.Itoa(sucCount)+" of "+strconv.Itoa(sucCount+errCount)+" Historic Update records created"))
}
