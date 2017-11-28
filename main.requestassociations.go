package main

import (
	"encoding/xml"
	"fmt"

	"github.com/hornbill/sqlx"
)

//processCallAssociations - Get all records from swdata.cmn_rel_opencall_oc, process accordingly
func processCallAssociations() {
	logger(1, "Processing Request Associations, please wait...", true)
	//Connect to the JSON specified DB
	db, err := sqlx.Open(appDBDriver, connStrAppDB)
	defer db.Close()
	if err != nil {
		logger(4, " [DATABASE] Database Connection Error for Request Associations: "+fmt.Sprintf("%v", err), false)
		return
	}
	//Check connection is open
	err = db.Ping()
	if err != nil {
		logger(4, " [DATABASE] [PING] Database Connection Error for Request Associations: "+fmt.Sprintf("%v", err), false)
		return
	}
	logger(3, "[DATABASE] Connection Successful", false)
	logger(3, "[DATABASE] Running query for Request Associations. Please wait...", false)

	//build query
	sqlDiaryQuery := swImportConf.RelatedRequestQuery
	logger(3, "[DATABASE] Request Association Query: "+sqlDiaryQuery, false)
	//Run Query
	rows, err := db.Queryx(sqlDiaryQuery)
	if err != nil {
		logger(4, " Database Query Error: "+fmt.Sprintf("%v", err), false)
		return
	}

	//Process each association record, insert in to Hornbill
	//fmt.Println("Maximum Request Association Go Routines:", maxGoroutines)
	maxGoroutinesGuard := make(chan struct{}, maxGoroutines)
	for rows.Next() {
		var requestRels reqRelStruct

		errDataMap := rows.StructScan(&requestRels)
		if errDataMap != nil {
			logger(4, " Data Mapping Error: "+fmt.Sprintf("%v", errDataMap), false)
			return
		}
		smMasterRef, mrOK := arrCallsLogged[requestRels.MasterRef]
		smSlaveRef, srOK := arrCallsLogged[requestRels.SlaveRef]

		maxGoroutinesGuard <- struct{}{}
		wgAssoc.Add(1)
		go func() {
			defer wgAssoc.Done()
			if mrOK == true && smMasterRef != "" && srOK == true && smSlaveRef != "" {
				//We have Master and Slave calls matched in the SM database
				addAssocRecord(smMasterRef, smSlaveRef)
			}
			<-maxGoroutinesGuard
		}()
	}
	wgAssoc.Wait()
	logger(1, "Request Association Processing Complete", true)
}

//addAssocRecord - given a Master Reference and a Slave Refernce, adds a call association record to Service Manager
func addAssocRecord(masterRef string, slaveRef string) {
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return
	}
	espXmlmc.SetParam("entityId", masterRef)
	espXmlmc.SetParam("entityName", "Requests")
	espXmlmc.SetParam("linkedEntityId", slaveRef)
	espXmlmc.SetParam("linkedEntityName", "Requests")
	espXmlmc.SetParam("updateTimeline", "true")
	espXmlmc.SetParam("visibility", "trustedGuest")
	XMLUpdate, xmlmcErr := espXmlmc.Invoke("apps/com.hornbill.servicemanager/RelationshipEntities", "add")
	if xmlmcErr != nil {
		//		log.Fatal(xmlmcErr)
		logger(4, "Unable to create Request Association between ["+masterRef+"] and ["+slaveRef+"] :"+fmt.Sprintf("%v", xmlmcErr), false)
		return
	}
	var xmlRespon xmlmcResponse
	errXMLMC := xml.Unmarshal([]byte(XMLUpdate), &xmlRespon)
	if errXMLMC != nil {
		logger(4, "Unable to read response from Hornbill instance for Request Association between ["+masterRef+"] and ["+slaveRef+"] :"+fmt.Sprintf("%v", errXMLMC), false)
		return
	}
	if xmlRespon.MethodResult != "ok" {
		logger(5, "Unable to add Request Association between ["+masterRef+"] and ["+slaveRef+"] : "+xmlRespon.State.ErrorRet, false)
		return
	}
	logger(1, "Request Association Success between ["+masterRef+"] and ["+slaveRef+"]", false)
}
