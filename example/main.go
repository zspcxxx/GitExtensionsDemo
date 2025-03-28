package main

import (
	"google.golang.org/grpc"
	"golang.org/x/net/context"
	"log"
	pb "Utility/pb/MQTTClient"
	pbmsg "Utility/pb/SAAMessage"
	"fmt"
)

func main()  {
	data1 := "{\"DeviceType\":\"TH-S100L\", \"DeviceSN\":\"ACCF23E1C8A7\", \"DeviceID\":9}"
	data1 := "{\"DeviceType\":\"TH-CFA600L\", \"DeviceSN\":\"B0F89315BFAE\", \"DeviceID\":15}"
	data1 := "{\"Topic\":\"TELLHOW/TH-S100L/ACCF23E1C8A7/TEST\", \"Payload\":\"Mqtt Pub test!\"}"
	data1 := "TELLHOW/TH-CFA600L/B0F89315BFAE/SET"
	data2 := "{\"ver\":1,\"tp\":21,\"data\":[{\"m\":\"FANFREQ\",\"v\":25,\"p\":3}],\"controlFrom\":\"ionic@test3@1531744313792@4032\"}"
	data2 := "{\\\"ver\\\":1,\\\"tp\\\":21,\\\"data\\\":[{\\\"m\\\":\\\"FANFREQ\\\",\\\"v\\\":25,\\\"p\\\":3}],\\\"controlFrom\\\":\\\"ionic@test3@1531744313792@4032\\\"}"
	data2 := "{\\\"Ver\\\":1,\\\"Tp\\\":21,\\\"Data\\\":[{\\\"M\\\":\\\"FANFREQ\\\",\\\"V\\\":25,\\\"P\\\":3}],\\\"ControlFrom\\\":\\\"ionic@test3@1531744313792@4032\\\"}"

	Data := fmt.Sprintf("{\"Topic\":\"%s\",\"Payload\":\"%s\"}", data1, data2)
	//conn, err := grpc.Dial("192.168.3.113:40600", grpc.WithInsecure())
	conn, err := grpc.Dial("192.168.3.192:40600", grpc.WithInsecure())

 	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewMQTTClientClient(conn)

	if err != nil {
		log.Println(err)
	}

	req := new(pbmsg.InvokeServiceRequest)
	req.Uid = 10
	req.Payload = Data
	//r, err := c.AddDevice(context.Background(), req)
	//r, err := c.RemoveDevice(context.Background(), req)
	r, err := c.PublishTopic(context.Background(), req)
	if err != nil {
		log.Fatalf("could not DBExecute: %v", err)
	}
	fmt.Printf("\r\n")
	fmt.Printf("\r\n")
	fmt.Printf("Payload = %s\r\n", r.Payload)

}
