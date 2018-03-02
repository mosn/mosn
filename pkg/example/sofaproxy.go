package main

import "gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
import (
	"bytes"
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/codec"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
)

func main() {

	//initilize codec engine. TODO: config driven
	codecImpl := codec.NewProtocols(map[byte]protocol.Protocol{
		sofarpc.PROTOCOL_CODE_V1:sofarpc.BoltV1,
		sofarpc.PROTOCOL_CODE_V2:sofarpc.BoltV2,
	})

	//plug-in tr codec
	codecImpl.PutProtocol(sofarpc.PROTOCOL_CODE, sofarpc.Tr)


	trPostData := bytes.NewBuffer([]byte("\x0d\x00TaobaoRemoting"))
	boltV1PostData := bytes.NewBuffer([]byte("\x01\x00BoltV1"))
	boltV2PostData := bytes.NewBuffer([]byte("\x02\x00BoltV2"))

	//test tr decode branch
	fmt.Println("-----------> tr test begin")
	codecImpl.Decode(nil, trPostData, nil)
	fmt.Println("<----------- tr test end\n")

	//test boltv1 decode branch
	fmt.Println("-----------> boltv1 test begin")
	codecImpl.Decode(nil, boltV1PostData, nil)
	fmt.Println("<----------- boltv1 test end\n")

	//test boltv2 decode branch
	fmt.Println("-----------> boltv2 test begin")
	codecImpl.Decode(nil, boltV2PostData, nil)
	fmt.Println("<----------- boltv2 test end\n")
}
