package main

import (
	"encoding/xml"
	"fmt"
	"sync"

	"github.com/hornbill/sqlx"
)

//processCallAssociations - Get all records from swdata.cmn_rel_opencall_oc, process accordingly
func processCallAssociations() {
	logger(1, "Processing Request Associations, please wait...", true)
	//Connect to the JSON specified DB
	db, err := sqlx.Open(appDBDriver, connStrAppDB)
	if err != nil {
		logger(4, " [DATABASE] Database Connection Error for Request Associations: "+fmt.Sprintf("%v", err), false)
		return
	}
	defer db.Close()
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

	var wg sync.WaitGroup

	jobs := make(chan refStruct, maxGoroutines)

	for w := 1; w <= maxGoroutines; w++ {
		go addAssocRecord(jobs, wg)
	}

	for rows.Next() {
		var requestRels reqRelStruct
		wg.Add(1)

		errDataMap := rows.StructScan(&requestRels)
		if errDataMap != nil {
			logger(4, " Data Mapping Error: "+fmt.Sprintf("%v", errDataMap), false)
			return
		}
		smMasterRef, mrOK := arrCallsLogged[requestRels.MasterRef]
		smSlaveRef, srOK := arrCallsLogged[requestRels.SlaveRef]

		if mrOK == true && smMasterRef != "" && srOK == true && smSlaveRef != "" {
			//We have Master and Slave calls matched in the SM database

			jobs <- refStruct{MasterRef: smMasterRef, SlaveRef: smSlaveRef}
		}
	}

	close(jobs)
	wg.Wait()

	logger(1, "Request Association Processing Complete", true)
}

//addAssocRecord - given a Master Reference and a Slave Refernce, adds a call association record to Service Manager
func addAssocRecord(jobs chan refStruct, wg sync.WaitGroup) {

	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		logger(4, "Could not connect to Hornbill Instance", false)
		wg.Done()
		return

	}

	for asset := range jobs {
		espXmlmc.SetParam("entityId", asset.MasterRef)
		espXmlmc.SetParam("entityName", "Requests")
		espXmlmc.SetParam("linkedEntityId", asset.SlaveRef)
		espXmlmc.SetParam("linkedEntityName", "Requests")
		espXmlmc.SetParam("updateTimeline", "true")
		espXmlmc.SetParam("visibility", "trustedGuest")
		XMLUpdate, xmlmcErr := espXmlmc.Invoke("apps/com.hornbill.servicemanager/RelationshipEntities", "add")
		if xmlmcErr != nil {
			//		log.Fatal(xmlmcErr)
			logger(4, "Unable to create Request Association between ["+asset.MasterRef+"] and ["+asset.SlaveRef+"] :"+fmt.Sprintf("%v", xmlmcErr), false)
			wg.Done()
			return
		}
		var xmlRespon xmlmcResponse
		errXMLMC := xml.Unmarshal([]byte(XMLUpdate), &xmlRespon)
		if errXMLMC != nil {
			logger(4, "Unable to read response from Hornbill instance for Request Association between ["+asset.MasterRef+"] and ["+asset.SlaveRef+"] :"+fmt.Sprintf("%v", errXMLMC), false)
			wg.Done()
			return
		}
		if xmlRespon.MethodResult != "ok" {
			logger(5, "Unable to add Request Association between ["+asset.MasterRef+"] and ["+asset.SlaveRef+"] : "+xmlRespon.State.ErrorRet, false)
			wg.Done()
			return
		}
		logger(1, "Request Association Success between ["+asset.MasterRef+"] and ["+asset.SlaveRef+"]", false)
		wg.Done()
	}
}
