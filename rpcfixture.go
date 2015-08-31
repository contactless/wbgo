package wbgo

import (
	"fmt"
	"github.com/stretchr/objx"
	"strconv"
	"strings"
	"testing"
)

const (
	SAMPLE_CLIENT_ID = "11111111"
)

type RpcFixture struct {
	*FakeMQTTFixture
	client     MQTTClient
	rpc        *MQTTRPCServer
	id         uint64
	app        string
	service    string
	clientName string
}

func NewRpcFixture(t *testing.T, app, service, clientName string, rcvr interface{}, methods ...string) (f *RpcFixture) {
	f = &RpcFixture{
		FakeMQTTFixture: NewFakeMQTTFixture(t),
		id:              1,
		app:             app,
		service:         service,
		clientName:      clientName,
	}
	f.rpc = NewMQTTRPCServer(app, f.Broker.MakeClient(clientName))
	f.rpc.Register(rcvr)
	f.client = f.Broker.MakeClient("tst")
	f.client.Start()
	f.rpc.Start()
	expect := []interface{}{
		fmt.Sprintf("Subscribe -- %s: /rpc/v1/%s/+/+/+", clientName, clientName),
	}
	for _, methodName := range methods {
		expect = append(expect, f.expectedMessage(clientName, methodName, "1", true))
	}
	f.Verify(expect...)
	return f
}

func (f *RpcFixture) topic(parts ...string) string {
	prefixParts := []string{"/rpc/v1", f.app, f.service}
	return strings.Join(append(prefixParts, parts...), "/")
}

func (f *RpcFixture) expectedMessage(from, subtopic, payload string, retained bool) string {
	rtn := ""
	if retained {
		rtn = ", retained"
	}
	return fmt.Sprintf("%s -> %s: [%s] (QoS 1%s)", from, f.topic(subtopic), payload, rtn)
}

func (f *RpcFixture) TearDownRPC() {
	f.rpc.Stop()
}

func (f *RpcFixture) verifyRpcRaw(subtopic string, params, expectedResponse objx.Map) {
	replyId := strconv.FormatUint(f.id, 10)
	request := objx.Map{
		"id":     replyId,
		"params": params,
	}
	f.id++
	subtopicWithId := subtopic + "/" + SAMPLE_CLIENT_ID
	payload := request.MustJSON()
	f.client.Publish(MQTTMessage{f.topic(subtopicWithId), payload, 1, false})
	resp := expectedResponse.Copy()
	resp["id"] = replyId
	f.Verify(
		f.expectedMessage("tst", subtopicWithId, payload, false),
		f.expectedMessage(f.clientName, subtopicWithId+"/reply", resp.MustJSON(), false))
}

func (f *RpcFixture) VerifyRpc(subtopic string, params objx.Map, expectedResult interface{}) {
	f.verifyRpcRaw(subtopic, params, objx.Map{"result": expectedResult})
}

func (f *RpcFixture) VerifyRpcError(subtopic string, param objx.Map, code int, typ string, msg string) {
	f.verifyRpcRaw(
		subtopic,
		param,
		objx.Map{
			"error": objx.Map{
				"message": msg,
				"code":    code,
				"data":    typ,
			},
		},
	)
}