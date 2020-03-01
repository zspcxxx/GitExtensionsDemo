package main


type ReturnJSON struct {
	Code int
	Data interface{}
}

type RequestData struct {
	DeviceType string
	DeviceSN string
	DeviceID int64
}

type PublishData struct {
	Topic string
	Payload string
}
//
//******************************
type DevResponseData struct {
	ALLProductInfo []*ProductInfo
	Count int64
}

type ProductInfo struct {
	DID int64
	ProductQrNum string
	MacNum string
	ProductType string
	ProductName string
	ScanInTime string
}
//******************************

//
//******************************
type AddDTRequestData struct {
	DID int64
	Data string
}
//******************************

//
//******************************
type DeviceInformation struct {
	DID int64
	ProductQrNum string
	MacNum string
	ProductType string
	ProductName string
	ScanInTime string
}
type AllDeviceInformation struct {
	ALLProductInfo []DeviceInformation
	Count int
}
type DeviceInformations struct {
	Code int
	Data AllDeviceInformation
}
//******************************

//
//******************************
type DataIdentify struct {
	Tp int
}
//******************************

//
//******************************
type DeviceReportInfoLevel struct {
	Ver string
	Tp int
	Id string
	Sn int64
	St int64
	Data []DeviceReportDataLevel
}

type DeviceReportDataLevel struct {
	K int64
	M string
	P int
	V string
}
//******************************

//
//******************************
type AddUpdateRequestData struct {
	DID int64
	NewData DeviceReportInfoLevel
}

type AddUpdateResponseData struct {
	ResponseData string
}
//******************************

//
//******************************
type HistrDataTime struct {
	Date string
	Hour int
}
//******************************

//
//******************************
type DeviceIntmedData struct {
	Data []DeviceReportDataLevel
}
//******************************

//
//******************************
type DeviceVerTwoRequest struct {
	DID int64
	NewData DeviceReportDataVerTwo
}
//******************************


//
//******************************
type DeviceReportDataVerTwo struct {
	Ver string
	D []DeviceUpdateData
}
//******************************

//
//******************************
type DeviceUpdateData struct {
	K int64
	V string
}
//******************************