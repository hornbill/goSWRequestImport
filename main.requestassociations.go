package main

import (
	"encoding/xml"
	"fmt"
)

//processCallAssociations - Get all records from swdata.cmn_rel_opencall_oc, process accordingly
func processCallAssociations() {
	logger(1, "Processing Request Associations, please wait...", true)
	//Check connection is open
	err := dbapp.Ping()
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
	rows, err := dbapp.Queryx(sqlDiaryQuery)
	if err != nil {
		logger(4, " Database Query Error: "+fmt.Sprintf("%v", err), false)
		return
	}

	for rows.Next() {
		var requestRels reqRelStruct

		errDataMap := rows.StructScan(&requestRels)
		if errDataMap != nil {
			logger(4, " Data Mapping Error: "+fmt.Sprintf("%v", errDataMap), false)
			return
		}
		smMasterRef, mrOK := arrCallsLogged[requestRels.MasterRef]
		smSlaveRef, srOK := arrCallsLogged[requestRels.SlaveRef]

		if mrOK == true && smMasterRef != "" && srOK == true && smSlaveRef != "" {
			//We have Master and Slave calls matched in the SM database
			jobs := refStruct{MasterRef: smMasterRef, SlaveRef: smSlaveRef}
			addAssocRecord(jobs)
		}
	}

	logger(1, "Request Association Processing Complete", true)
}

//addAssocRecord - given a Master Reference and a Slave Refernce, adds a call association record to Service Manager
func addAssocRecord(assoc refStruct) {

	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		logger(4, "Could not connect to Hornbill Instance", false)
		return

	}

	espXmlmc.SetParam("entityId", assoc.MasterRef)
	espXmlmc.SetParam("entityName", "Requests")
	espXmlmc.SetParam("linkedEntityId", assoc.SlaveRef)
	espXmlmc.SetParam("linkedEntityName", "Requests")
	espXmlmc.SetParam("updateTimeline", "true")
	espXmlmc.SetParam("visibility", "trustedGuest")
	XMLUpdate, xmlmcErr := espXmlmc.Invoke("apps/com.hornbill.servicemanager/RelationshipEntities", "add")
	if xmlmcErr != nil {
		//		log.Fatal(xmlmcErr)
		logger(4, "Unable to create Request Association between ["+assoc.MasterRef+"] and ["+assoc.SlaveRef+"] :"+fmt.Sprintf("%v", xmlmcErr), false)
		return
	}
	var xmlRespon xmlmcResponse
	errXMLMC := xml.Unmarshal([]byte(XMLUpdate), &xmlRespon)
	if errXMLMC != nil {
		logger(4, "Unable to read response from Hornbill instance for Request Association between ["+assoc.MasterRef+"] and ["+assoc.SlaveRef+"] :"+fmt.Sprintf("%v", errXMLMC), false)
		return
	}
	if xmlRespon.MethodResult != "ok" {
		logger(5, "Unable to add Request Association between ["+assoc.MasterRef+"] and ["+assoc.SlaveRef+"] : "+xmlRespon.State.ErrorRet, false)
		return
	}
	logger(1, "Request Association Success between ["+assoc.MasterRef+"] and ["+assoc.SlaveRef+"]", false)
}
