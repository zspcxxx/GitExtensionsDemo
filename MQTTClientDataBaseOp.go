package main

import (
	dah "Utility/DAHUtility"
	log "Utility/ZapLog"
	pb "Utility/pb/DataAccessHandler"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	THIRTYSEC = 30
	THIRTYMIN = MINUTE*30
	MINUTE = 60
)

const (
	THCOMP = -26
)

//历史数据更新项目
func histrDataOption (option string) (flag bool) {
	if (strings.Compare(option,"INDOORTH") == 0) {      //TH-CFA600L
		//室内温度
		return true
	} else if (strings.Compare(option,"OUTDOORTH") == 0) {
		//室外温度
		return true
	} else if (strings.Compare(option,"CO2CONC") == 0) {
		//CO2浓度
		return true
	} else if (strings.Compare(option,"PM25") == 0) {
		//PM25浓度
		return true
	} else if (strings.Compare(option,"OUTPARTCONC") == 0) {
		//室外颗粒物浓度
		return true
	} else if (strings.Compare(option,"HUMSENSOR") == 0) {
		//室内湿度
		return true
	} else if (strings.Compare(option,"PM10") == 0) {   //TH-S100L
		//PM10浓度
		return true
	} else if (strings.Compare(option,"TH") == 0) {   //TH-S100L
		//室内温度
		return true
	} else if (strings.Compare(option,"RH") == 0) {   //TH-S100L
		//室内湿度
		return true
	} else {
		return false
	}
}

//获取设备信息
func GetDeviceInfo() (*DevResponseData, error) {
	response := new(DevResponseData)

	q := dah.NewQueryHelper("DeviceInfo")
	q.SetType(pb.SqlExcuteType_SELECTCOUNT)
	q.Select("DID")
	q.And("Status", pb.Operator_EQUAL, 1)
	r, err := dah.Query(q)
	if err != nil {
		log.ErrorS("GetDeviceInfo", "dah.Query", err)
		return response,err
	}
	if len(r.Records) <= 0 {
		return response,err
	} else {
		response.Count = r.Records[0].GetInt64("Count")
	}

	query := dah.NewQueryHelper("DeviceInfo")
	query.Limit(0,response.Count)
	query.Select("DID", "DeviceQrNumber", "MACaddr", "DeviceType", "RecordTime", "Status","TypeInQRcode")
	query.And("Status", pb.Operator_EQUAL, 1)
	resp, err := dah.Query(query)
	if err != nil{
		log.ErrorS("GetDeviceInfo", "dah.Query", err)
		return response, err
	}
	deviceList := make([]*ProductInfo, 0)
	for _, value := range resp.Records {
		deviceInfo := new(ProductInfo)
		deviceInfo.DID = value.GetInt64("DID")
		deviceInfo.ProductQrNum = value.GetString("DeviceQrNumber")
		deviceInfo.MacNum = value.GetString("MACaddr")
		deviceInfo.ProductType = value.GetString("TypeInQRcode")
		deviceInfo.ScanInTime = value.GetString("RecordTime")
		deviceList = append(deviceList, deviceInfo)
	}
	response.ALLProductInfo = deviceList

	return response, nil
}


//新旧数据对比 更新实时数
func DeviceDataAddUpdate(request AddUpdateRequestData) ( error) {
	var oldData string
	var projectID int64
	devOldData := new(DeviceIntmedData)

	query := dah.NewQueryHelper("CurrentDeviceData")
	query.SetType(pb.SqlExcuteType_SELECT)
	query.Select("DID", "CurrentData","PID")
	query.And("DID", pb.Operator_EQUAL, request.DID)
	resp, err := dah.Query(query)
	if err != nil{
		log.ErrorS("DeviceDataAddUpdate", "dah.Query", err)
		return  err
	}
	for _, value := range resp.Records {
		oldData = value.GetString("CurrentData")
		projectID = value.GetInt64("PID")
	}


	if oldData == "" {
		for _, newValue := range request.NewData.Data{
			devOldData.Data = append(devOldData.Data, newValue)
		}
	}else{
		if err := json.Unmarshal([]byte(oldData), devOldData);
			err != nil{
			log.ErrorS("DeviceDataAddUpdate", "json.Unmarshal", err)
		}

		var fieldFlag bool
		for _, newValue := range request.NewData.Data {
			for i, oldValue := range devOldData.Data {
				if newValue.K == oldValue.K {
					devOldData.Data[i].V = newValue.V
					fieldFlag = true
				}
			}
			if fieldFlag == false {
				devOldData.Data = append(devOldData.Data, newValue)
			}
			fieldFlag = false
		}

	}
	b, _ := json.Marshal(devOldData)

	//更新实时数据
	timeNow := time.Now().Format("2006-01-02 15:04:05")
	datas := make([]*dah.ExecuteHelper, 0)//允许批量操作
	update := dah.NewExecuteHelper("`CurrentDeviceData`")
	update.SetType(pb.SqlExcuteType_UPDATE)
	update.AddData("CurrentData", string(b))
	update.AddData("ModifyTime", timeNow)
	update.And("DID", pb.Operator_EQUAL, request.DID)
	datas = append(datas, update)
	_, err = dah.Execute(datas)
	if err != nil {
		log.ErrorS("DeviceDataInput", "dah.Execute", err)
		return err
	}

	//redis update GetCurrentData
	dID := strconv.FormatInt(request.DID,10)
	key := dID + "|" + "GetCurrentData"
	inquiry,err := RedisCon.HSetStr(key,"CurrentData", string(b))
	if err != nil {
		log.ErrorS("DeviceDataInput Redis update fail", "RedisCon.HSet", err)
	}else{
		log.InfoS("DeviceDataInput Redis update Success","RedisCon.HSet",inquiry)
		RedisCon.ExpireKey(key,THIRTYSEC)
	}

	//redis Project Mean data update
	if projectID !=0 {
		pID := strconv.FormatInt(projectID,10)
		dID = strconv.FormatInt(request.DID,10)
		for _, insertValue := range devOldData.Data{
			//compare, _ := strconv.ParseInt(insertValue.V, 10, 64)
			key := pID + "|"+insertValue.M+"|GetProjectMean"
			if insertValue.M == "TH"{
				compare, _ := strconv.ParseInt(insertValue.V, 10, 64)
				//温度最多到零下二十六
				if compare > THCOMP{

					inquiry ,err := RedisCon.HSetStr(key,dID,insertValue.V)
					if err != nil {
						log.ErrorS("DeviceDataAddUpdate redis failed", "RedisCon.HSetStr", err)
					}else{
						log.InfoS("DeviceDataAddUpdate redis success", "RedisCon.HSetStr", inquiry)
					}
				}
			}else{
				if !strings.Contains(insertValue.V,"-") {
					inquiry ,err := RedisCon.HSetStr(key,dID,insertValue.V)
					if err != nil {
						log.ErrorS("DeviceDataAddUpdate redis failed", "RedisCon.HSetStr", err)
					}else{
						log.InfoS("DeviceDataAddUpdate redis success", "RedisCon.HSetStr", inquiry)

					}
				}
			}
			RedisCon.ExpireKey(key,THIRTYSEC)
		}
	}

	return nil
}

func readLatestAddDevice(n *int64) error {
	var tmpDID int64
	var tmpTopic string
	query := dah.NewQueryHelper("DeviceInfo")
	query.SetType(pb.SqlExcuteType_SELECT)
	query.Select("DID", "MACaddr", "DeviceType", "Status","PID")
	query.And("DID", pb.Operator_GT, *n)
	query.And("Status", pb.Operator_EQUAL, 1)
	resp, err := dah.Query(query)
	if err != nil{
		log.ErrorS("readLatestAddDevice", "dah.Query", err)
		return err
	}
	for _, value := range resp.Records {
		tmpDID = value.GetInt64("DID")

		if tmpDID > *n {
			*n = tmpDID
		}
		tmpTopic = fmt.Sprintf("TELLHOW/%s/%s/COV", value.GetString("DeviceType"), value.GetString("MACaddr"))
		tmptopicMap[tmpTopic] = tmpDID
	}
	return nil
}

func DeviceDataUpdateVerTwo(devRequest DeviceVerTwoRequest) error{
	var oldData string
	devOldData := new(DeviceReportDataVerTwo)
	devOldData.Ver = SECONDVERSION

	query := dah.NewQueryHelper("CurrentDeviceData")
	query.SetType(pb.SqlExcuteType_SELECT)
	query.Select("DID", "CurrentData")
	query.And("DID", pb.Operator_EQUAL, devRequest.DID)
	resp, err := dah.Query(query)
	if err != nil{
		log.ErrorS("DeviceDataUpdateVerTwo", "dah.Query", err)
		return  err
	}
	for _, value := range resp.Records {
		oldData = value.GetString("CurrentData")
	}


	if oldData == "" {
		for _, newValue := range devRequest.NewData.D{
			devOldData.D = append(devOldData.D, newValue)
		}
	}else{
		if err := json.Unmarshal([]byte(oldData), devOldData);
			err != nil {
			log.ErrorS("DeviceDataUpdateVerTwo", "json.Unmarshal", err)
			return err
		}

		var fieldFlag bool
		for _, newValue := range devRequest.NewData.D {
			for i, oldValue := range devOldData.D {
				if newValue.K == oldValue.K {
					if newValue.K == 100 {
						newValue.V = checkRHValue(newValue.V)
					}
					devOldData.D[i].V = newValue.V
					fieldFlag = true
				}
			}
			if fieldFlag == false {
				devOldData.D = append(devOldData.D, newValue)
			}
			fieldFlag = false
		}
	}

	b, _ := json.Marshal(devOldData)

	//更新实时数据
	timeNow := time.Now().Format("2006-01-02 15:04:05")
	datas := make([]*dah.ExecuteHelper, 0)//允许批量操作
	update := dah.NewExecuteHelper("`CurrentDeviceData`")
	update.SetType(pb.SqlExcuteType_UPDATE)
	update.AddData("CurrentData", string(b))
	update.AddData("ModifyTime", timeNow)
	update.And("DID", pb.Operator_EQUAL, devRequest.DID)
	datas = append(datas, update)
	_, err = dah.Execute(datas)
	if err != nil {
		log.ErrorS("DeviceDataInput", "dah.Execute", err)
		return err
	}

	//redis update GetCurrentData
	dID := strconv.FormatInt(devRequest.DID,10)
	key := dID + "|" + "GetCurrentData"
	inquiry,err := RedisCon.HSetStr(key,"CurrentData", string(b))
	if err != nil {
		log.ErrorS("DeviceDataInput Redis update fail", "RedisCon.HSet", err)
	}else{
		log.InfoS("DeviceDataInput Redis update Success","RedisCon.HSet",inquiry)
		RedisCon.ExpireKey(key,THIRTYSEC)
	}

	return nil
}

func checkRHValue (value string) string {

	rhValue, _ := strconv.ParseInt(value, 10, 64)
	if rhValue > 100 || rhValue < 0 {
		value = "-255"
	}

	return value
}