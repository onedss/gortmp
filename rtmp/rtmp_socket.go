package rtmp

import (
	"errors"
	"github.com/onedss/gortmp/util"
	//"fmt"
	"io"
	//"reflect"
)

const (
	SEND_CHUNK_SIZE_MESSAGE         = "Send Chunk Size Message"
	SEND_ACK_MESSAGE                = "Send Acknowledgement Message"
	SEND_ACK_WINDOW_SIZE_MESSAGE    = "Send Window Acknowledgement Size Message"
	SEND_SET_PEER_BANDWIDTH_MESSAGE = "Send Set Peer Bandwidth Message"

	SEND_STREAM_BEGIN_MESSAGE       = "Send Stream Begin Message"
	SEND_SET_BUFFER_LENGTH_MESSAGE  = "Send Set Buffer Lengh Message"
	SEND_STREAM_IS_RECORDED_MESSAGE = "Send Stream Is Recorded Message"

	SEND_PING_REQUEST_MESSAGE  = "Send Ping Request Message"
	SEND_PING_RESPONSE_MESSAGE = "Send Ping Response Message"

	SEND_CONNECT_MESSAGE          = "Send Connect Message"
	SEND_CONNECT_RESPONSE_MESSAGE = "Send Connect Response Message"

	SEND_CREATE_STREAM_MESSAGE          = "Send Create Stream Message"
	SEND_CREATE_STREAM_RESPONSE_MESSAGE = "Send Create Stream Response Message"

	SEND_PLAY_MESSAGE          = "Send Play Message"
	SEND_PLAY_RESPONSE_MESSAGE = "Send Play Response Message"

	SEND_PUBLISH_RESPONSE_MESSAGE = "Send Publish Response Message"
	SEND_PUBLISH_START_MESSAGE    = "Send Publish Start Message"

	SEND_UNPUBLISH_RESPONSE_MESSAGE = "Send Unpublish Response Message"

	SEND_AUDIO_MESSAGE      = "Send Audio Message"
	SEND_FULL_AUDIO_MESSAGE = "Send Full Audio Message"
	SEND_VIDEO_MESSAGE      = "Send Video Message"
	SEND_FULL_VDIEO_MESSAGE = "Send Full Video Message"
)

func newConnectResponseMessageData(objectEncoding float64) (amfobj AMFObjects) {
	amfobj = newAMFObjects()
	amfobj["fmsVer"] = "Donview/1.0"
	amfobj["capabilities"] = 31
	amfobj["mode"] = 1
	amfobj["Author"] = "Donview"
	amfobj["level"] = Level_Status
	amfobj["code"] = NetConnection_Connect_Success
	amfobj["objectEncoding"] = uint64(objectEncoding)

	return
}

func newPublishResponseMessageData(streamid uint32, code, level string) (amfobj AMFObjects) {
	amfobj = newAMFObjects()
	amfobj["code"] = code
	amfobj["level"] = level
	amfobj["streamid"] = streamid

	return
}

func newPlayResponseMessageData(streamid uint32, code, level string) (amfobj AMFObjects) {
	amfobj = newAMFObjects()
	amfobj["code"] = code
	amfobj["level"] = level
	amfobj["streamid"] = streamid

	return
}

func recvMessage(conn *RtmpNetConnection) (msg RtmpMessage, err error) {
	if conn.readSeqNum >= conn.bandwidth {
		conn.totalRead += conn.readSeqNum
		conn.readSeqNum = 0
		//sendAck(conn, conn.totalRead)
		sendMessage(conn, SEND_ACK_MESSAGE, conn.totalRead)
	}

	msg, err = readChunk(conn)
	if err != nil {
		return nil, err
	}

	// ??????????????????????????????????????????,?????????????????????????????????????????????,
	// ?????????????????????????????????.??????????????????????????????,????????????????????????.
	messageType := msg.Header().ChunkMessgaeHeader.MessageTypeID
	if RTMP_MSG_CHUNK_SIZE <= messageType && messageType <= RTMP_MSG_EDGE {
		switch messageType {
		case RTMP_MSG_CHUNK_SIZE:
			{
				m := msg.(*ChunkSizeMessage)
				conn.readChunkSize = int(m.ChunkSize)
				return recvMessage(conn)
			}
		case RTMP_MSG_ABORT:
			{
				m := msg.(*AbortMessage)
				delete(conn.incompleteRtmpBody, m.ChunkStreamId)
				return recvMessage(conn)
			}
		case RTMP_MSG_ACK:
			{
				return recvMessage(conn)
			}
		case RTMP_MSG_USER_CONTROL:
			{
				if _, ok := msg.(*PingRequestMessage); ok {
					//sendPingResponse(conn)
					sendMessage(conn, SEND_PING_RESPONSE_MESSAGE, nil)
				}
				return recvMessage(conn)
			}
		case RTMP_MSG_ACK_SIZE:
			{
				m := msg.(*WindowAcknowledgementSizeMessage)
				conn.bandwidth = m.AcknowledgementWindowsize
				return recvMessage(conn)
			}
		case RTMP_MSG_BANDWIDTH:
			{
				m := msg.(*SetPeerBandwidthMessage)
				conn.bandwidth = m.AcknowledgementWindowsize
				return recvMessage(conn)
			}
		case RTMP_MSG_EDGE:
			{
				return recvMessage(conn)
			}
		}
	}

	return msg, err
}

func sendMessage(conn *RtmpNetConnection, message string, args interface{}) error {
	switch message {
	case SEND_CHUNK_SIZE_MESSAGE:
		{
			size, ok := args.(uint32)
			if !ok {
				return errors.New(SEND_CHUNK_SIZE_MESSAGE + ", The parameter only one(size uint32)!")
			}

			m := newChunkSizeMessage()
			m.ChunkSize = size
			m.Encode()
			head := newRtmpHeader(RTMP_CSID_CONTROL, 0, uint32(len(m.RtmpBody.Payload)), RTMP_MSG_CHUNK_SIZE, 0, 0)
			m.RtmpHeader = head
			return writeMessage(conn, m)
		}
	case SEND_ACK_MESSAGE:
		{
			num, ok := args.(uint32)
			if !ok {
				return errors.New(SEND_ACK_MESSAGE + ", The parameter only one(number uint32)!")
			}

			m := newAcknowledgementMessage()
			m.SequenceNumber = num
			m.Encode()
			head := newRtmpHeader(RTMP_CSID_CONTROL, 0, uint32(len(m.RtmpBody.Payload)), RTMP_MSG_ACK, 0, 0)
			m.RtmpHeader = head
			return writeMessage(conn, m)
		}
	case SEND_ACK_WINDOW_SIZE_MESSAGE:
		{
			size, ok := args.(uint32)
			if !ok {
				return errors.New(SEND_ACK_WINDOW_SIZE_MESSAGE + ", The parameter only one(size uint32)!")
			}

			m := newWindowAcknowledgementSizeMessage()
			m.AcknowledgementWindowsize = size
			m.Encode()
			head := newRtmpHeader(RTMP_CSID_CONTROL, 0, uint32(len(m.RtmpBody.Payload)), RTMP_MSG_ACK_SIZE, 0, 0)
			m.RtmpHeader = head
			return writeMessage(conn, m)
		}
	case SEND_SET_PEER_BANDWIDTH_MESSAGE:
		{
			size, ok := args.(uint32)
			if !ok {
				return errors.New(SEND_SET_PEER_BANDWIDTH_MESSAGE + ", The parameter only one(size uint32)!")
			}

			m := newSetPeerBandwidthMessage()
			m.AcknowledgementWindowsize = size
			m.LimitType = byte(2) // Dynamic
			m.Encode()
			head := newRtmpHeader(RTMP_CSID_CONTROL, 0, uint32(len(m.RtmpBody.Payload)), RTMP_MSG_BANDWIDTH, 0, 0)
			m.RtmpHeader = head
			return writeMessage(conn, m)
		}
	case SEND_STREAM_BEGIN_MESSAGE:
		{
			if args != nil {
				return errors.New(SEND_STREAM_BEGIN_MESSAGE + ", The parameter is nil")
			}

			m := newStreamBeginMessage()
			m.EventType = RTMP_USER_STREAM_BEGIN
			m.StreamID = conn.streamID
			m.Encode()
			head := newRtmpHeader(RTMP_CSID_CONTROL, 0, uint32(len(m.RtmpBody.Payload)), RTMP_MSG_USER_CONTROL, 0, 0)
			m.RtmpHeader = head
			return writeMessage(conn, m)
		}
	case SEND_STREAM_IS_RECORDED_MESSAGE:
		{
			if args != nil {
				return errors.New(SEND_STREAM_IS_RECORDED_MESSAGE + ", The parameter is nil")
			}

			m := newStreamIsRecordedMessage()
			m.EventType = RTMP_USER_STREAM_IS_RECORDED
			m.StreamID = conn.streamID
			m.Encode()
			head := newRtmpHeader(RTMP_CSID_CONTROL, 0, uint32(len(m.RtmpBody.Payload)), RTMP_MSG_USER_CONTROL, 0, 0)
			m.RtmpHeader = head
			return writeMessage(conn, m)
		}
	case SEND_SET_BUFFER_LENGTH_MESSAGE:
		{
			if args != nil {
				return errors.New(SEND_SET_BUFFER_LENGTH_MESSAGE + ", The parameter is nil")
			}

			m := newSetBufferMessage()
			m.EventType = RTMP_USER_SET_BUFFLEN
			m.StreamID = conn.streamID
			m.Millisecond = 100
			m.Encode()
			head := newRtmpHeader(RTMP_CSID_CONTROL, 0, uint32(len(m.RtmpBody.Payload)), RTMP_MSG_USER_CONTROL, 0, 0)
			m.RtmpHeader = head
			return writeMessage(conn, m)
		}
	case SEND_PING_REQUEST_MESSAGE:
		{
			if args != nil {
				return errors.New(SEND_PING_REQUEST_MESSAGE + ", The parameter is nil")
			}

			m := newPingRequestMessage()
			m.EventType = RTMP_USER_PING_REQUEST
			m.Encode()
			head := newRtmpHeader(RTMP_CSID_CONTROL, 0, uint32(len(m.RtmpBody.Payload)), RTMP_MSG_USER_CONTROL, 0, 0)
			m.RtmpHeader = head
			return writeMessage(conn, m)
		}
	case SEND_PING_RESPONSE_MESSAGE:
		{
			if args != nil {
				return errors.New(SEND_PING_RESPONSE_MESSAGE + ", The parameter is nil")
			}

			m := newPingResponseMessage()
			m.EventType = RTMP_USER_PING_RESPONSE
			m.Encode()
			head := newRtmpHeader(RTMP_CSID_CONTROL, 0, uint32(len(m.RtmpBody.Payload)), RTMP_MSG_USER_CONTROL, 0, 0)
			m.RtmpHeader = head
			return writeMessage(conn, m)
		}
	case SEND_CREATE_STREAM_MESSAGE:
		{
			if args != nil {
				return errors.New(SEND_CREATE_STREAM_MESSAGE + ", The parameter is nil")
			}

			m := newCreateStreamMessage()
			m.CommandName = "createStream"
			m.TransactionId = 1
			m.Encode0()
			head := newRtmpHeader(RTMP_CSID_COMMAND, 0, uint32(len(m.RtmpBody.Payload)), RTMP_MSG_AMF0_COMMAND, 0, 0)
			m.RtmpHeader = head
			return writeMessage(conn, m)
		}
	case SEND_CREATE_STREAM_RESPONSE_MESSAGE:
		{
			tid, ok := args.(uint64)
			if !ok {
				return errors.New(SEND_CREATE_STREAM_RESPONSE_MESSAGE + ", The parameter only one(TransactionId uint64)!")
			}

			m := newResponseCreateStreamMessage()
			m.CommandName = Response_Result
			m.TransactionId = tid
			m.StreamId = conn.streamID
			m.Encode0()
			head := newRtmpHeader(RTMP_CSID_COMMAND, 0, uint32(len(m.RtmpBody.Payload)), RTMP_MSG_AMF0_COMMAND, 0, 0)
			m.RtmpHeader = head
			return writeMessage(conn, m)
		}
	case SEND_PLAY_MESSAGE:
		{
			data, ok := args.(map[interface{}]interface{})
			if !ok {
				errors.New(SEND_PLAY_MESSAGE + ", The parameter is map[interface{}]interface{}")
			}

			var streamName string
			var start uint64
			var duration uint64
			var rest bool

			for i, v := range data {
				if i == "StreamName" {
					streamName = v.(string)
				} else if i == "Start" {
					start = v.(uint64)
				} else if i == "Duration" {
					duration = v.(uint64)
				} else if i == "Rest" {
					rest = v.(bool)
				}
			}

			m := newPlayMessage()
			m.CommandName = "play"
			m.TransactionId = 1
			m.StreamName = streamName
			m.Start = start
			m.Duration = duration
			m.Rest = rest
			m.Encode0()
			head := newRtmpHeader(RTMP_CSID_COMMAND, 0, uint32(len(m.RtmpBody.Payload)), RTMP_MSG_AMF0_COMMAND, 0, 0)
			m.RtmpHeader = head
			return writeMessage(conn, m)
		}
	case SEND_PLAY_RESPONSE_MESSAGE:
		{
			data, ok := args.(AMFObjects)
			if !ok {
				errors.New(SEND_PLAY_RESPONSE_MESSAGE + ", The parameter is AMFObjects(map[string]interface{})")
			}

			obj := newAMFObjects()
			var streamID uint32

			for i, v := range data {
				switch i {
				case "code":
					{
						obj[i] = v
					}
				case "level":
					{
						obj[i] = v
					}
				case "streamid":
					{
						if t, ok := v.(uint32); ok {
							streamID = t
						}
					}
				}
			}

			obj["clientid"] = 1

			m := newResponsePlayMessage()
			m.CommandName = Response_OnStatus
			m.TransactionId = 0
			m.Object = obj
			m.Encode0()
			head := newRtmpHeader(RTMP_CSID_COMMAND, 0, uint32(len(m.RtmpBody.Payload)), RTMP_MSG_AMF0_COMMAND, streamID, 0)
			m.RtmpHeader = head
			return writeMessage(conn, m)
		}
	case SEND_CONNECT_RESPONSE_MESSAGE:
		{
			data, ok := args.(AMFObjects)
			if !ok {
				errors.New(SEND_CONNECT_RESPONSE_MESSAGE + ", The parameter is AMFObjects(map[string]interface{})")
			}

			pro := newAMFObjects()
			info := newAMFObjects()

			for i, v := range data {
				switch i {
				case "fmsVer":
					{
						pro[i] = v
					}
				case "capabilities":
					{
						pro[i] = v
					}
				case "mode":
					{
						pro[i] = v
					}
				case "Author":
					{
						pro[i] = v
					}
				case "level":
					{
						info[i] = v
					}
				case "code":
					{
						info[i] = v
					}
				case "objectEncoding":
					{
						info[i] = v
					}
				}
			}

			m := newResponseConnectMessage()
			m.CommandName = Response_Result
			m.TransactionId = 1
			m.Properties = pro
			m.Infomation = info
			m.Encode0()
			head := newRtmpHeader(RTMP_CSID_COMMAND, 0, uint32(len(m.RtmpBody.Payload)), RTMP_MSG_AMF0_COMMAND, 0, 0)
			m.RtmpHeader = head
			return writeMessage(conn, m)
		}
	case SEND_CONNECT_MESSAGE:
		{
			data, ok := args.(AMFObjects)
			if !ok {
				errors.New(SEND_CONNECT_MESSAGE + ", The parameter is AMFObjects(map[string]interface{})")
			}

			obj := newAMFObjects()
			info := newAMFObjects()

			for i, v := range data {
				switch i {
				case "app":
					{
						obj[i] = v
					}
				case "audioCodecs":
					{
						obj[i] = v
					}
				case "videoCodecs":
					{
						obj[i] = v
					}
				case "tcUrl":
					{
						obj[i] = v
					}
				case "swfUrl":
					{
						obj[i] = v
					}
				case "pageUrl":
					{
						obj[i] = v
					}
				case "capabilities":
					{
						obj[i] = v
					}
				case "flashVer":
					{
						obj[i] = v
					}
				case "fpad":
					{
						obj[i] = v
					}
				case "objectEncoding":
					{
						obj[i] = v
					}
				case "videoFunction":
					{
						obj[i] = v
					}
				}

			}

			m := newConnectMessage()
			m.CommandName = "connect"
			m.TransactionId = 1
			m.Object = obj
			m.Optional = info
			m.Encode0()
			head := newRtmpHeader(RTMP_CSID_COMMAND, 0, uint32(len(m.RtmpBody.Payload)), RTMP_MSG_AMF0_COMMAND, 0, 0)
			m.RtmpHeader = head
			return writeMessage(conn, m)
		}
	case SEND_PUBLISH_RESPONSE_MESSAGE, SEND_PUBLISH_START_MESSAGE:
		{
			data, ok := args.(AMFObjects)
			if !ok {
				errors.New(SEND_CONNECT_MESSAGE + "or" + SEND_PUBLISH_START_MESSAGE + ", The parameter is AMFObjects(map[string]interface{})")
			}

			info := newAMFObjects()
			var streamID uint32

			for i, v := range data {
				switch i {
				case "code":
					{
						info[i] = v
					}
				case "level":
					{
						info[i] = v
					}
				case "streamid":
					{
						if t, ok := v.(uint32); ok {
							streamID = t
						}
					}
				}
			}

			info["clientid"] = 1

			m := newResponsePublishMessage()
			m.CommandName = Response_OnStatus
			m.TransactionId = 0
			m.Infomation = info
			m.Encode0()
			head := newRtmpHeader(RTMP_CSID_COMMAND, 0, uint32(len(m.RtmpBody.Payload)), RTMP_MSG_AMF0_COMMAND, streamID, 0)
			m.RtmpHeader = head
			return writeMessage(conn, m)
		}
	case SEND_UNPUBLISH_RESPONSE_MESSAGE:
		{
		}
	case SEND_FULL_AUDIO_MESSAGE:
		{
			audio, ok := args.(*AVPacket)
			if !ok {
				errors.New(SEND_FULL_AUDIO_MESSAGE + ", The parameter is AVPacket")
			}

			return sendAVMessage(conn, audio, true, true)
		}
	case SEND_AUDIO_MESSAGE:
		{
			audio, ok := args.(*AVPacket)
			if !ok {
				errors.New(SEND_AUDIO_MESSAGE + ", The parameter is AVPacket")
			}

			return sendAVMessage(conn, audio, true, false)
		}
	case SEND_FULL_VDIEO_MESSAGE:
		{
			video, ok := args.(*AVPacket)
			if !ok {
				errors.New(SEND_FULL_VDIEO_MESSAGE + ", The parameter is AVPacket")
			}

			return sendAVMessage(conn, video, false, true)
		}
	case SEND_VIDEO_MESSAGE:
		{
			video, ok := args.(*AVPacket)
			if !ok {
				errors.New(SEND_VIDEO_MESSAGE + ", The parameter is AVPacket")
			}

			return sendAVMessage(conn, video, false, false)
		}
	}

	return errors.New("send message no exist")
}

func writeMessage(conn *RtmpNetConnection, msg RtmpMessage) error {
	if conn.writeSeqNum > conn.bandwidth {
		conn.totalWrite += conn.writeSeqNum
		conn.writeSeqNum = 0
		sendMessage(conn, SEND_ACK_MESSAGE, conn.totalWrite)
		sendMessage(conn, SEND_PING_REQUEST_MESSAGE, nil)
	}

	mark, need, err := encodeChunk12(msg.Header(), msg.Body().Payload, conn.writeChunkSize)
	if err != nil {
		return err
	}

	_, err = conn.bw.Write(mark)
	if err != nil {
		return err
	}

	err = conn.bw.Flush()
	if err != nil {
		return err
	}

	conn.writeSeqNum += uint32(len(mark))

	for need != nil && len(need) > 0 {
		mark, need, err = encodeChunk1(msg.Header(), need, conn.writeChunkSize)
		if err != nil {
			return err
		}

		_, err = conn.bw.Write(mark)
		if err != nil {
			return err
		}

		err = conn.bw.Flush()
		if err != nil {
			return err
		}

		conn.writeSeqNum += uint32(len(mark))
	}

	return nil
}

// ?????????????????????????????????,???????????????12?????????,Chunk Message Header???????????????TimeStamp,??????????????????
// ???????????????4,8?????????,Chunk Message Header???????????????TimeStamp Delta,??????????????????Chunk???????????????
// ???????????????0?????????,Chunk Message Header??????????????????,????????????Chunk???????????????
func sendAVMessage(conn *RtmpNetConnection, av *AVPacket, isAudio bool, isFirst bool) error {
	if conn.writeSeqNum > conn.bandwidth {
		conn.totalWrite += conn.writeSeqNum
		conn.writeSeqNum = 0
		sendMessage(conn, SEND_ACK_MESSAGE, conn.totalWrite)
		sendMessage(conn, SEND_PING_REQUEST_MESSAGE, nil)
	}

	var err error
	var mark []byte
	var need []byte
	var head *RtmpHeader

	if isAudio {
		head = newRtmpHeader(RTMP_CSID_AUDIO, av.Timestamp, uint32(len(av.Payload)), RTMP_MSG_AUDIO, conn.streamID, 0)
	} else {
		head = newRtmpHeader(RTMP_CSID_VIDEO, av.Timestamp, uint32(len(av.Payload)), RTMP_MSG_VIDEO, conn.streamID, 0)
	}

	// ???????????????????????????,????????????????????????(Chunk Basic Header(1) + Chunk Message Header(11) + Extended Timestamp(4)(??????????????????))
	// ????????????,?????????????????????????????????,??????????????????,?????????????????????(Chunk Basic Header(1) + Chunk Message Header(7))
	// ???Chunk Type???0???(???Chunk12),
	if isFirst {
		mark, need, err = encodeChunk12(head, av.Payload, conn.writeChunkSize)
	} else {
		mark, need, err = encodeChunk8(head, av.Payload, conn.writeChunkSize)
	}

	if err != nil {
		return err
	}

	_, err = conn.bw.Write(mark)
	if err != nil {
		return err
	}

	err = conn.bw.Flush()
	if err != nil {
		return err
	}

	conn.writeSeqNum += uint32(len(mark))

	// ???????????????????????????,??????????????????,???????????????????????????(data + Chunk Basic Header(1))
	for need != nil && len(need) > 0 {
		mark, need, err = encodeChunk1(head, need, conn.writeChunkSize)
		if err != nil {
			return err
		}

		_, err = conn.bw.Write(mark)
		if err != nil {
			return err
		}

		err = conn.bw.Flush()
		if err != nil {
			return err
		}

		conn.writeSeqNum += uint32(len(mark))
	}

	return nil
}

func readChunk(conn *RtmpNetConnection) (msg RtmpMessage, err error) {
	head, err := conn.br.ReadByte()
	conn.readSeqNum += 1
	if err != nil {
		return nil, err
	}

	cbh := new(ChunkBasicHeader)
	cbh.ChunkStreamID = uint32(head & 0x3f) // 0011 1111
	cbh.ChunkType = (head & 0xc0) >> 6      // 1100 0000

	// ????????????ID???0,1??????,???????????????.
	cbh.ChunkStreamID, err = readChunkStreamID(conn, cbh.ChunkStreamID)
	if err != nil {
		return nil, errors.New("get chunk stream id error :" + err.Error())
	}

	if conn.rtmpHeader[cbh.ChunkStreamID] == nil {
		//conn.rtmpHeader[cbh.ChunkStreamID] = &RtmpHeader{ChunkBasicHeader.ChunkType: cbh.ChunkType, ChunkBasicHeader.ChunkStreamID: cbh.ChunkStreamID}
		conn.rtmpHeader[cbh.ChunkStreamID] = &RtmpHeader{ChunkBasicHeader: ChunkBasicHeader{ChunkType: cbh.ChunkType, ChunkStreamID: cbh.ChunkStreamID}}
	}

	h := conn.rtmpHeader[cbh.ChunkStreamID]
	if cbh.ChunkType != 3 && conn.incompleteRtmpBody[cbh.ChunkStreamID] != nil {
		// ?????????????????????3,????????????rtmp???body????????????.
		return nil, errors.New("incompleteRtmpBody error")
	}

	chunkHead, err := readChunkType(conn, h, cbh.ChunkType)
	if err != nil {
		return nil, errors.New("get chunk type error :" + err.Error())
	}

	if conn.incompleteRtmpBody[cbh.ChunkStreamID] == nil {
		conn.incompleteRtmpBody[cbh.ChunkStreamID] = make([]byte, 0)
	}

	markRead := uint32(len(conn.incompleteRtmpBody[cbh.ChunkStreamID]))
	needRead := uint32(conn.readChunkSize)
	unRead := chunkHead.ChunkMessgaeHeader.MessageLength - markRead
	if unRead < needRead {
		needRead = unRead
	}

	buf := make([]byte, needRead)
	n, err := io.ReadFull(conn.br, buf)
	if err != nil {
		return nil, err
	}

	conn.readSeqNum += uint32(n)

	buf = append(conn.incompleteRtmpBody[cbh.ChunkStreamID], buf...)
	conn.incompleteRtmpBody[cbh.ChunkStreamID] = buf

	// ?????????????????????????????????,???????????????????????????,???????????????????????????.
	if uint32(len(conn.incompleteRtmpBody[cbh.ChunkStreamID])) == chunkHead.ChunkMessgaeHeader.MessageLength {

		rtmpHeader := chunkHead.Clone()
		//rtmpBody := conn.incompleteRtmpBody[cbh.ChunkStreamID]
		rtmpBody := new(RtmpBody)
		rtmpBody.Payload = conn.incompleteRtmpBody[cbh.ChunkStreamID]

		msg = GetRtmpMessage(rtmpHeader, rtmpBody)

		delete(conn.incompleteRtmpBody, cbh.ChunkStreamID)

		return msg, nil
	}

	return readChunk(conn)
}

func readChunkStreamID(conn *RtmpNetConnection, csid uint32) (chunkStreamID uint32, err error) {
	switch csid {
	case 0:
		{
			u8, err := conn.br.ReadByte()
			conn.readSeqNum += 1
			if err != nil {
				return 0, err
			}

			chunkStreamID = 64 + uint32(u8)
		}
	case 1:
		{
			u16 := make([]byte, 2)
			if _, err = io.ReadFull(conn.br, u16); err != nil {
				return
			}

			conn.readSeqNum += 2
			chunkStreamID = 64 + uint32(u16[0]) + 256*uint32(u16[1])
		}
	}

	chunkStreamID = csid

	return chunkStreamID, nil
}

func readChunkType(conn *RtmpNetConnection, h *RtmpHeader, chunkType byte) (head *RtmpHeader, err error) {
	switch chunkType {
	case 0:
		{
			// Timestamp 3 bytes
			b := make([]byte, 3)
			if _, err := io.ReadFull(conn.br, b); err != nil {
				return nil, err
			}
			conn.readSeqNum += 3
			h.ChunkMessgaeHeader.Timestamp = util.BigEndian.Uint24(b) //type = 0???????????????????????????,???????????????????????????

			// Message Length 3 bytes
			if _, err = io.ReadFull(conn.br, b); err != nil { // ??????Message Length,????????????????????????????????????????????????????????????????????????????????????,?????????Chunk data?????????.
				return nil, err
			}
			conn.readSeqNum += 3
			h.ChunkMessgaeHeader.MessageLength = util.BigEndian.Uint24(b)

			// Message Type ID 1 bytes
			v, err := conn.br.ReadByte() // ??????Message Type ID
			if err != nil {
				return nil, err
			}
			conn.readSeqNum += 1
			h.ChunkMessgaeHeader.MessageTypeID = v

			// Message Stream ID 4bytes
			bb := make([]byte, 4)
			if _, err = io.ReadFull(conn.br, bb); err != nil { // ??????Message Stream ID
				return nil, err
			}
			conn.readSeqNum += 4
			h.ChunkMessgaeHeader.MessageStreamID = util.LittleEndian.Uint32(bb)

			// ExtendTimestamp 4 bytes
			if h.ChunkMessgaeHeader.Timestamp == 0xffffff { // ??????type 0???chunk,??????????????????????????????,??????????????????????????????0xffffff(16777215),???????????????0xffffff,????????????????????????????????????,????????????????????????
				if _, err = io.ReadFull(conn.br, bb); err != nil {
					return nil, err
				}
				conn.readSeqNum += 4
				h.ChunkExtendedTimestamp.ExtendTimestamp = util.BigEndian.Uint32(bb)
			}
		}
	case 1:
		{
			// Timestamp 3 bytes
			b := make([]byte, 3)
			if _, err = io.ReadFull(conn.br, b); err != nil {
				return nil, err
			}
			conn.readSeqNum += 3
			h.ChunkBasicHeader.ChunkType = chunkType
			h.ChunkMessgaeHeader.Timestamp = util.BigEndian.Uint24(b)

			// Message Length 3 bytes
			if _, err = io.ReadFull(conn.br, b); err != nil {
				return nil, err
			}
			conn.readSeqNum += 3
			h.ChunkMessgaeHeader.MessageLength = util.BigEndian.Uint24(b)

			// Message Type ID 1 bytes
			v, err := conn.br.ReadByte()
			if err != nil {
				return nil, err
			}
			conn.readSeqNum += 1
			h.ChunkMessgaeHeader.MessageTypeID = v

			// ExtendTimestamp 4 bytes
			if h.ChunkMessgaeHeader.Timestamp == 0xffffff {
				bb := make([]byte, 4)
				if _, err := io.ReadFull(conn.br, bb); err != nil {
					return nil, err
				}
				conn.readSeqNum += 4
				h.ChunkExtendedTimestamp.ExtendTimestamp = util.BigEndian.Uint32(bb)
			}
		}
	case 2:
		{
			// Timestamp 3 bytes
			b := make([]byte, 3)
			if _, err = io.ReadFull(conn.br, b); err != nil {
				return nil, err
			}
			conn.readSeqNum += 3
			h.ChunkBasicHeader.ChunkType = chunkType
			h.ChunkMessgaeHeader.Timestamp = util.BigEndian.Uint24(b)

			// ExtendTimestamp 4 bytes
			if h.ChunkMessgaeHeader.Timestamp == 0xffffff {
				bb := make([]byte, 4)
				if _, err := io.ReadFull(conn.br, bb); err != nil {
					return nil, err
				}
				conn.readSeqNum += 4
				h.ChunkExtendedTimestamp.ExtendTimestamp = util.BigEndian.Uint32(bb)
			}
		}
	case 3:
		{
			h.ChunkBasicHeader.ChunkType = chunkType
		}
	}

	head = h
	return head, nil
}
