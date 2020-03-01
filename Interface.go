package main

import (
	pb "Utility/pb/SAAMessage"
	"context"
	log "Utility/ZapLog"
	"encoding/json"
	"fmt"
)

const (
	//error code
	OK = 0
	INTERNALERRORCODE = 100
	INPUTPARAMSERRORCODE = 101
)

type server struct {}

func newServer() *server {
	s := new(server)
	return s
}

func generateJSON(errCode int, data interface{}) string {
	ret := new(ReturnJSON)
	ret.Code = errCode
	ret.Data = data
	b, err := json.Marshal(ret)
	if err != nil{
		log.ErrorS("generateJSON", "json.Marshal", err)
	}

	return string(b)
}

func (s *server) AddDevice(ctx context.Context, in *pb.InvokeServiceRequest) (*pb.InvokeServiceResponse, error) {
	payload := in.GetPayload()
	log.InfoS("AddDevice", "Get Payload", payload)
	var request RequestData
	err := json.Unmarshal([]byte(payload), &request)
	if err != nil {
		log.ErrorS("AddDevice", "Unmarshal error", err)
		return &pb.InvokeServiceResponse{Payload: generateJSON(INPUTPARAMSERRORCODE, nil)}, nil
	}
	topic := fmt.Sprintf(covTopic, request.DeviceType, request.DeviceSN)
	addTopicMap(topic, request.DeviceID)
	err = subscribeTopic0(topic)
	if err != nil{
		log.ErrorS("AddDevice", "subscribeTopic0", err)
		return &pb.InvokeServiceResponse{Payload: generateJSON(INTERNALERRORCODE, nil)}, nil
	}
	return &pb.InvokeServiceResponse{Payload: generateJSON(OK, nil)}, nil
}

func (s *server) RemoveDevice(ctx context.Context, in *pb.InvokeServiceRequest) (*pb.InvokeServiceResponse, error) {
	payload := in.GetPayload()
	log.InfoS("RemoveDevice", "Get Payload", payload)
	var request RequestData
	err := json.Unmarshal([]byte(payload), &request)
	if err != nil {
		log.ErrorS("RemoveDevice", "Unmarshal error", err)
		return &pb.InvokeServiceResponse{Payload: generateJSON(INPUTPARAMSERRORCODE, nil)}, nil
	}
	topic := fmt.Sprintf(covTopic, request.DeviceType, request.DeviceSN)
	err = unsubscribeTopic(topic)
	if err != nil{
		log.ErrorS("RemoveDevice", "unsubscribeTopic", err)
		return &pb.InvokeServiceResponse{Payload: generateJSON(INTERNALERRORCODE, nil)}, nil
	}
	removeTopicMap(topic)
	return &pb.InvokeServiceResponse{Payload: generateJSON(OK, nil)}, nil
}

func (s *server) PublishTopic(ctx context.Context, in *pb.InvokeServiceRequest) (*pb.InvokeServiceResponse, error) {
	payload := in.GetPayload()
	log.InfoS("PublishTopic", "Get Payload", payload)
	var request PublishData
	err := json.Unmarshal([]byte(payload), &request)
	if err != nil {
		log.ErrorS("PublishTopic", "Unmarshal error", err)
		return &pb.InvokeServiceResponse{Payload: generateJSON(INPUTPARAMSERRORCODE, nil)}, nil
	}
	err = publishTopic0(request)
	if err != nil{
		log.ErrorS("PublishTopic", "publishTopic0", err)
		return &pb.InvokeServiceResponse{Payload: generateJSON(INTERNALERRORCODE, nil)}, nil
	}
	return &pb.InvokeServiceResponse{Payload: generateJSON(OK, nil)}, nil
}