package registry

import (
    "time"
    "net"
    "fmt"
    "gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/model"
    "github.com/golang/protobuf/proto"
    "encoding/binary"
)

func MockRpcServer() {
    go run()
}

func run() {
    l, _ := net.Listen("tcp", "127.0.0.1:8089")

    defer l.Close()

    for {
        select {
        case <-time.After(2 * time.Second):
            conn, _ := l.Accept()

            fmt.Printf("[REALSERVER]get connection %s..", conn.RemoteAddr())
            fmt.Println()

            buf := make([]byte, 4*1024)

            for {
                t := time.Now()
                conn.SetReadDeadline(t.Add(3 * time.Second))

                if bytesRead, err := conn.Read(buf); err != nil {

                    if err, ok := err.(net.Error); ok && err.Timeout() {
                        continue
                    }

                    fmt.Println("[REALSERVER]failed read buf")
                    return
                } else {
                    request := decodeBoltRequest(buf[:bytesRead])
                    fmt.Println("------------Received Data---------- ")
                    fmt.Printf("BoltReqId = %d", request.ReqId)
                    fmt.Println()
                    publishRequestPb := &model.PublisherRegisterPb{}
                    err := proto.Unmarshal(request.Content, publishRequestPb)
                    var regId string
                    if err == nil {
                        fmt.Println("Recievied publish request.")
                        fmt.Println("DataId: " + publishRequestPb.BaseRegister.DataId)
                        fmt.Printf("Data: %v", publishRequestPb.DataList)
                        regId = publishRequestPb.BaseRegister.RegistId

                        fmt.Println()
                        conn.Write(assembleRegisterResponse(regId, request.ReqId))

                    } else {
                        subscriberRequestPb := &model.SubscriberRegisterPb{}
                        err := proto.Unmarshal(request.Content, subscriberRequestPb)
                        if err == nil {
                            fmt.Println("Recievied subscriber request.")
                            fmt.Println("Scope = " + subscriberRequestPb.Scope)
                            fmt.Println("DataId = " + subscriberRequestPb.BaseRegister.DataId)
                            regId = subscriberRequestPb.BaseRegister.RegistId
                            fmt.Println("RegId = " + regId)
                            //do response
                            fmt.Println()
                            conn.Write(assembleRegisterResponse(regId, request.ReqId))

                            //write data
                            time.Sleep(1 * time.Second)
                            fmt.Println("Write data...")
                            receivedDataCmd := assembleReceivedDataRequest(subscriberRequestPb.BaseRegister.DataId)
                            conn.Write(doEncodeRequestCommand(receivedDataCmd))
                        }
                    }

                    //time.Sleep(4 * time.Second)
                    //break

                }
            }
        }
    }

}

func assembleRegisterResponse(registId string, boltReqId uint32) []byte {
    class := "com.alipay.sofa.registry.core.model"
    response := &model.RegisterResponsePb{
        Success:  true,
        RegistId: registId,
        Version:  100,
        Refused:  false,
        Message:  "",
    }

    resBytes, _ := proto.Marshal(response)

    bolt := &sofarpc.BoltResponseCommand{
        Protocol:       1,
        CmdType:        0,
        CmdCode:        2,
        Version:        1,
        ReqId:          boltReqId,
        CodecPro:       11,
        ResponseStatus: 0,
        ClassLen:       int16(len(class)),
        HeaderLen:      0,
        ContentLen:     len(resBytes),
        ClassName:      []byte(class),
        HeaderMap:      make([]byte, 0, 0),
        Content:        resBytes,
    }
    return doEncodeResponseCommand(bolt)
}

func decodeBoltRequest(bytes []byte) *sofarpc.BoltRequestCommand {

    read := 0
    dataType := bytes[1]

    cmdCode := binary.BigEndian.Uint16(bytes[2:4])
    ver2 := bytes[4]
    requestId := binary.BigEndian.Uint32(bytes[5:9])
    codec := bytes[9]
    timeout := binary.BigEndian.Uint32(bytes[10:14])
    classLen := binary.BigEndian.Uint16(bytes[14:16])
    headerLen := binary.BigEndian.Uint16(bytes[16:18])
    contentLen := binary.BigEndian.Uint32(bytes[18:22])

    read = sofarpc.REQUEST_HEADER_LEN_V1
    var class, header, content []byte

    if classLen > 0 {
        class = bytes[read: read+int(classLen)]
        read += int(classLen)
    }
    if headerLen > 0 {
        header = bytes[read: read+int(headerLen)]
        read += int(headerLen)
    }
    if contentLen > 0 {
        content = bytes[read: read+int(contentLen)]
        read += int(contentLen)
    }

    return &sofarpc.BoltRequestCommand{

        Protocol:   sofarpc.PROTOCOL_CODE_V1,
        CmdType:    dataType,
        CmdCode:    int16(cmdCode),
        Version:    ver2,
        ReqId:      requestId,
        CodecPro:   codec,
        Timeout:    int(timeout),
        ClassLen:   int16(classLen),
        HeaderLen:  int16(headerLen),
        ContentLen: int(contentLen),
        ClassName:  class,
        HeaderMap:  header,
        Content:    content,
    }

}

var v int64 = 0

func assembleReceivedDataRequest(dataId string) *sofarpc.BoltRequestCommand {
    v ++
    dataBox := &model.DataBoxesPb{
        Data: []*model.DataBoxPb{{"data1"}, {"data2"}, {"data3"}},
    }

    dataBox2 := &model.DataBoxesPb{
        Data: []*model.DataBoxPb{{"c1"}, {"c2"}, {"c3"}},
    }

    rd := &model.ReceivedDataPb{
        DataId:  dataId,
        Segment: "s1",
        Data:    map[string]*model.DataBoxesPb{"zone1": dataBox, "zone2": dataBox2},
        Version: v,
    }

    class := "com.alipay.confreg"
    data, _ := proto.Marshal(rd)

    return &sofarpc.BoltRequestCommand{
        Protocol:   sofarpc.PROTOCOL_CODE_V1,
        CmdType:    1,
        CmdCode:    1,
        Version:    1,
        ReqId:      114,
        CodecPro:   11,
        Timeout:    int(3000),
        ClassLen:   int16(len(class)),
        HeaderLen:  int16(0),
        ContentLen: int(len(data)),
        ClassName:  []byte(class),
        HeaderMap:  nil,
        Content:    data,
    }
}

func doEncodeResponseCommand(cmd *sofarpc.BoltResponseCommand) []byte {

    var data []byte

    data = append(data, cmd.Protocol, cmd.CmdType)
    cmdCodeBytes := make([]byte, 2)
    binary.BigEndian.PutUint16(cmdCodeBytes, uint16(cmd.CmdCode))
    data = append(data, cmdCodeBytes...)
    data = append(data, cmd.Version)

    requestIdBytes := make([]byte, 4)
    binary.BigEndian.PutUint32(requestIdBytes, uint32(cmd.ReqId))
    data = append(data, requestIdBytes...)
    data = append(data, cmd.CodecPro)

    respStatusBytes := make([]byte, 2)
    binary.BigEndian.PutUint16(respStatusBytes, uint16(cmd.ResponseStatus))
    data = append(data, respStatusBytes...)

    clazzLengthBytes := make([]byte, 2)
    binary.BigEndian.PutUint16(clazzLengthBytes, uint16(cmd.ClassLen))
    data = append(data, clazzLengthBytes...)

    headerLengthBytes := make([]byte, 2)
    binary.BigEndian.PutUint16(headerLengthBytes, uint16(cmd.HeaderLen))
    data = append(data, headerLengthBytes...)

    contentLenBytes := make([]byte, 4)
    binary.BigEndian.PutUint32(contentLenBytes, uint32(cmd.ContentLen))
    data = append(data, contentLenBytes...)

    if cmd.ClassLen > 0 {
        data = append(data, cmd.ClassName...)
    }

    if cmd.HeaderLen > 0 {
        data = append(data, cmd.HeaderMap...)
    }
    if cmd.ContentLen > 0 {
        data = append(data, cmd.Content...)
    }

    return data
}

func doEncodeRequestCommand(cmd *sofarpc.BoltRequestCommand) []byte {
    var data []byte

    data = append(data, cmd.Protocol, cmd.CmdType)
    cmdCodeBytes := make([]byte, 2)
    binary.BigEndian.PutUint16(cmdCodeBytes, uint16(cmd.CmdCode))
    data = append(data, cmdCodeBytes...)
    data = append(data, cmd.Version)

    requestIdBytes := make([]byte, 4)
    binary.BigEndian.PutUint32(requestIdBytes, uint32(cmd.ReqId))
    data = append(data, requestIdBytes...)
    data = append(data, cmd.CodecPro)

    timeoutBytes := make([]byte, 4)
    binary.BigEndian.PutUint32(timeoutBytes, uint32(cmd.Timeout))
    data = append(data, timeoutBytes...)

    clazzLengthBytes := make([]byte, 2)
    binary.BigEndian.PutUint16(clazzLengthBytes, uint16(cmd.ClassLen))
    data = append(data, clazzLengthBytes...)

    headerLengthBytes := make([]byte, 2)
    binary.BigEndian.PutUint16(headerLengthBytes, uint16(cmd.HeaderLen))
    data = append(data, headerLengthBytes...)

    contentLenBytes := make([]byte, 4)
    binary.BigEndian.PutUint32(contentLenBytes, uint32(cmd.ContentLen))
    data = append(data, contentLenBytes...)

    if cmd.ClassLen > 0 {
        data = append(data, cmd.ClassName...)
    }

    if cmd.HeaderLen > 0 {
        data = append(data, cmd.HeaderMap...)
    }

    if cmd.ContentLen > 0 {
        data = append(data, cmd.Content...)
    }

    return data
}
