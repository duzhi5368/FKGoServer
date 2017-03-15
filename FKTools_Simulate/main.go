//---------------------------------------------
package main

//---------------------------------------------
import (
	DH "FKGoServer/FKLib_Common/DH"
	MSGDEFINE "FKGoServer/FKLib_Common/MsgDefine"
	PACKET "FKGoServer/FKLib_Common/Packet"
	RC4 "crypto/rc4"
	BINARY "encoding/binary"
	FMT "fmt"
	IO "io"
	LOG "log"
	BIG "math/big"
	RAND "math/rand"
	NET "net"
	OS "os"
	TIME "time"
)

//---------------------------------------------
var (
	seqid        = uint32(0)
	encoder      *RC4.Cipher
	decoder      *RC4.Cipher
	KEY_EXCHANGE = false
	SALT         = "DH"
)

//---------------------------------------------
const (
	DEFAULT_AGENT_HOST = "192.168.2.127:8888"
)

//---------------------------------------------
func checkErr(err error) {
	if err != nil {
		LOG.Println(err)
		panic("error occured in protocol module")
	}
}

//---------------------------------------------
func main() {
	host := DEFAULT_AGENT_HOST
	if env := OS.Getenv("AGENT_HOST"); env != "" {
		host = env
	}
	addr, err := NET.ResolveTCPAddr("tcp", host)
	if err != nil {
		LOG.Println(err)
		OS.Exit(-1)
	}
	conn, err := NET.DialTCP("tcp", nil, addr)
	if err != nil {
		LOG.Println(err)
		OS.Exit(-1)
	}
	defer conn.Close()

	//get_seed_req
	S1, M1 := DH.DHExchange()
	S2, M2 := DH.DHExchange()
	p2 := MSGDEFINE.S_seed_info{
		int32(M1.Int64()),
		int32(M2.Int64()),
	}
	rst := send_proto(conn, MSGDEFINE.Code["get_seed_req"], p2)
	r1, _ := MSGDEFINE.PKT_seed_info(rst)
	LOG.Printf("result: %#v", r1)

	K1 := DH.DHKey(S1, BIG.NewInt(int64(r1.F_client_send_seed)))
	K2 := DH.DHKey(S2, BIG.NewInt(int64(r1.F_client_receive_seed)))
	encoder, err = RC4.NewCipher([]byte(FMT.Sprintf("%v%v", SALT, K1)))
	if err != nil {
		LOG.Println(err)
		return
	}
	decoder, err = RC4.NewCipher([]byte(FMT.Sprintf("%v%v", SALT, K2)))
	if err != nil {
		LOG.Println(err)
		return
	}

	KEY_EXCHANGE = true

	//user_login_req
	p3 := MSGDEFINE.S_user_login_info{
		F_login_way:          0,
		F_open_udid:          "udid",
		F_client_certificate: "qwertyuiopasdfgh",
		F_client_version:     1,
		F_user_lang:          "en",
		F_app_id:             "com.yrhd.lovegame",
		F_os_version:         "android4.4",
		F_device_name:        "simulate",
		F_device_id:          "device_id",
		F_device_id_type:     1,
		F_login_ip:           "127.0.0.1",
	}
	send_proto(conn, MSGDEFINE.Code["user_login_req"], p3)

	//heart_beat_req
	p4 := MSGDEFINE.S_auto_id{
		F_id: RAND.Int31(),
	}
	send_proto(conn, MSGDEFINE.Code["heart_beat_req"], p4)

	//proto_ping_req
	p5 := MSGDEFINE.S_auto_id{
		F_id: RAND.Int31(),
	}
	send_proto(conn, MSGDEFINE.Code["proto_ping_req"], p5)

}

//---------------------------------------------
func send_proto(conn NET.Conn, p int16, info interface{}) (reader *PACKET.Packet) {
	seqid++
	payload := PACKET.Func_Pack(p, info, nil)
	writer := PACKET.Writer()
	writer.WriteU16(uint16(len(payload)) + 4)

	w := PACKET.Writer()
	w.WriteU32(seqid)
	w.WriteRawBytes(payload)
	data := w.Data()
	if KEY_EXCHANGE {
		encoder.XORKeyStream(data, data)
	}
	writer.WriteRawBytes(data)
	conn.Write(writer.Data())
	LOG.Printf("send : %#v", writer.Data())
	TIME.Sleep(TIME.Second)

	//read
	header := make([]byte, 2)
	IO.ReadFull(conn, header)
	size := BINARY.BigEndian.Uint16(header)
	r := make([]byte, size)
	_, err := IO.ReadFull(conn, r)
	if err != nil {
		LOG.Println(err)
	}
	if KEY_EXCHANGE {
		decoder.XORKeyStream(r, r)
	}
	reader = PACKET.Reader(r)
	b, err := reader.ReadS16()
	if err != nil {
		LOG.Println(err)
	}
	if _, ok := MSGDEFINE.RCode[b]; !ok {
		LOG.Println("unknown proto ", b)
	}

	return
}

//---------------------------------------------
