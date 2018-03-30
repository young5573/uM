/*******************************
   JMPSYSTEMS uManager v3.3
   2016.12.25 by bb

   <사이트 설치시 변경 필요>
   DB url : www.umanager.kr    3306
   Web Server url : www.umanager.kr     80, 8080, 8100
*******************************/
package main

import (
	//syslog "github.com/RackSec/srslog"
	"encoding/base64"
	"net"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"gopkg.in/mcuadros/go-syslog.v2"
	"log"
	"net/http"
	"runtime"
	"time"
	"sync"
)
import (
	"github.com/tatsushid/go-fastping"
	"bufio"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"os"
	"strconv"
	"strings"
	// "strconv"
	"crypto/md5"
	"github.com/gorilla/websocket"
	"io"

	"archive/zip"
	"compress/flate"
	"html/template"
	"io/ioutil"
	"os/exec"
	"path"
	"path/filepath"
	//"debug/macho"
	"encoding/hex"
	"encoding/xml"

)

/***********	웹소켓	*****************/
// 원래 웹소켓
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024 * 10000,
	WriteBufferSize: 1024 * 10000,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
var runos string
var userid,userpw string //로그인 유저 아이디 패스워드 (디비에 저장됨)
var topCounters struct {
	downloadTraffic uint64
	uploadTraffic   uint64
	totalUsers      uint64
	serviceTime     uint64
	activationRate  uint64
	wiredStatus     uint64
}
var licensekey = "jmpsystem.com"
var table string
var spend_time int
var dbrow string
var gSystemLockStatus string // 시스템 락 해제 여부
var gWebCnt int              // 웹소켓 들어와서 무한루프 카운팅..
var gApCnt int               // AP 갯수
var webSocVal_A05 int        // 마지막 카운터 값
var comparetime int          //index1에서 상태확인해줄 값 bootflag로는 장비가 죽은지 확인이 불가능
var gProg string             //  펌웨어 업데이트 하는 도중을 알려줌. 1,2,3,4...
var newmonthtable int
var newmonthap int
var debug_level string
var inputQueryString string
var pingpong ="NO"

//var osfile *os.File

var webSocString string     // 관리패킷 웹소켓으로 전송하기 위해 모으는 곳
var webSocString_Rec string //새로운 관리 패널에서 사각형 안에 들어갈 스트링 모으는 곳
var webSocString_Rec_time string
var webSocString_map string
var gMsgSentCnt int // 관리패킷 웹에 보여주는 용도. 웹소켓에 보낼 갯수 알려주기..

var tsuccess string //index2에 보낼 json파일 만드는 전역변수
var pktime string
var nmac1 string
var nmac2 string
var nmac3 string
var countwebsoc int

/***********	시간에 대한 go package 지식	*****************
	// secs := time.Now().Unix() // 유닉스 초 받아오기 --> 1481779113
	// fmt.Println(secs)
	// DB에서 읽어서 일반적인 형식으로 표시할때 :: 시간예 1470975817
	// 유닉스 초를 날짜로 바꾸기 --> 2016-12-15 14:18:33 +0900 KST
	// fmt.Println(time.Unix(secs, 0))

// mySQL DB에서 사용하는 unsigned int 사용시 유닉스 타임 초 충분히 소화 가능함. (4.2G 개.. 42억개) Go 언어 int 는 너무 많이 남고..
****************************************************************/

/************	인터페이스 에 대한 조사	******************
인터페이스는 메서드들의 모음으로 간단히 정의 할 수 있다. 또한 그 자체로 하나의 타입이기도 하다.
메서드들의 형태만 정의하고, 구현은 외부에 맡기는 방식으로 유연한 코드를 만들 수 있다.
*************************************************************/

/************	커멘트 창에서 아그 받아오기 	***************
만일 자주 변경하여 시험할 코드일 경우 아그들을 cmd 창에서 입력받도록 해보자
   이때 flag를 사용하면 명령창에 옵션명, 기본값, 설명순으로 flag 구조체를 만들고 Parse 를 때리면 한방에 들어간다.
   옵션이 없음을 찾아낼 수도 있다
port := flag.String("p", "8100", "port to serve on")
directory := flag.String("d", ".", "the directory of static file to host")
flag.Parse()
if flag.NFlag() == 0 { // 명령줄 옵션의 개수가 0개이면
		flag.Usage()   // 명령줄 옵션 기본 사용법 출력
		return
	}
***************************************************************/

/*************	문서 통째로 스트링 변수화 	*****************
` ` 으로 둘러싸면 여러줄에 걸쳐 있고 줄바꿈(엔터)까지 그대로 string 변수에 넣을 수 있다
var query_text_example string
query_text_example =
	`bootFlag=0 & macAddr=00:06:7a:e7:20 & ipAddr=14.71.128.227 & btVer=-mt76xx-bt.16.12.24 & fwVer=m7700-fw.16.12.21" + "& ssidW0=Airces_2G & ssidW1=Aircess_5G & chanW0=11 & chanW1=149"
	"& rxByteW0=12237134 & txByteW0=24000124450 & rxByteW1=3598714030, txByteW1=4609003000"
	"& rxPktW0=15043 & txPktW0=23400 & rxPktW1=381920 & txPktW1=482012 & assoCtW0=2 & assoCtW1=35"
	"& devTemp=42 & curMem=64000000 & pingTime=72 & assoDevT=1470975817 & leaveDevT=1470978888`
fmt.Println(query_text_example)
****************************************************************/

/*
//curl & 포스트맨 & 브라우져 쿼리 시험용 text :: [주의] 공백도 인식하므로 딱딱 붙여써야 한다 !!
bootFlag=0&macAddr=00:06:7a:e7:20&ipAddr=14.71.128.227&btVer=mt76xx-bt.16.12.24&fwVer=m7700-fw.16.12.21&ssidW0=Airces_2G&ssidW1=Aircess_5G&chanW0=11&chanW1=149&rxByteW0=12237134&txByteW0=24000124450&rxByteW1=3598714030&txByteW1=4609003000&rxPktW0=15043&txPktW0=23400&rxPktW1=381920&txPktW1=482012&assoCtW0=2&assoCtW1=35&devTemp=42&curMem=64000000&pingTime=72&assoDevT=1470975817&leaveDevT=1470978888
*/
const apTableFieldDefine string = " ( " +
	"id         INT(10) UNSIGNED, " + // [서버내부용] 서버에서 사용 자체 관리변수로 증가시킴. 아직 큰 의미 없음
	"time       INT(10) UNSIGNED, " + // [서버내부용] 유닉스 초 기록

	"bootFlag   TINYINT(1) unsigned," + // AP는 부팅직후 첫 패킷 보낼때 1로, 그외는 0 : 재부팅 이벤트 잡을때 사용
	"macAddr    varchar(17), " + // AP 라벨에 있는 MAC 주소 = 무선콘솔 MAC 주소 : 00:06:7A:22:0A:70
	"ipAddr     varchar(15), " + // AP 브리지 IP주소, 게이트웨이 모드일때는 WAN ip 주소 : 214:78:91:124
	"btVer      varchar(30), " + // 모델명 기호, 부트버전 : "m77-bt.16.12.24"
	"fwVer      varchar(18), " + // 모델명 기호, 펌버전   : "m7700-fw.16.12.21"

	"ssidW0     varchar(60), " + // SSID 들  20->40으로 변경함
	"ssidW1     varchar(60), " +
	"chanW0     varchar(3), " + // 채널번호 1~14 ... 36~169
	"chanW1     varchar(3), " +

	"rxByteW0   bigint(11) unsigned, " + // 바이트 카운터. 수천억경까지 가능
	"txByteW0   bigint(11) unsigned, " +
	"rxByteW1   bigint(11) unsigned, " +
	"txByteW1   bigint(11) unsigned, " +

	"rxPktW0    bigint(11) unsigned, " + // 패킷 카운터. 수천억경까지 가능
	"txPktW0    bigint(11) unsigned, " +
	"rxPktW1    bigint(11) unsigned, " +
	"txPktW1    bigint(11) unsigned, " +

	"assoCtW0   int(3) unsigned, " + // 무선0에 대한 접속된 무선 단말수
	"assoCtW1   int(3) unsigned, " +

	"devTemp    TINYINT(2), " + // 온도 -127~+127도
	"curMem     int(10) unsigned, " + // 0~6210300 : free 바이트 값으로 4GB 까지 표시 가능
	"pingTime   int(5) unsigned, " + // AP가 conf 값 url 쪽으로 2초마다 ping 날린 값 평균 응답시간 : 123 (msec) // 핑이 (5)회이상 끊어지면 0으로 기록

	"assoDevT   int(10) unsigned, " + // 단말 붙었을때 시각을 실시간 전송. 유닉스 초
	"leaveDevT  int(10) unsigned, " + // 단말 떨어질때 시각을 실시간 전송. 유닉스 초... 마지막에 쩜 없어야..
	"peerAp0 varchar(17), " +
	"peerAp1 varchar(17), " +
	"staMac0 varchar(17) " +
	")"

const apTableInsert string = "(id, time, bootFlag, macAddr, ipAddr, btVer, fwVer, ssidW0, ssidW1, chanW0, chanW1, " + // 11 개
	"rxByteW0, txByteW0, rxByteW1, txByteW1, rxPktW0, txPktW0, rxPktW1, txPktW1, assoCtW0, " + //  9 개
	"assoCtW1, devTemp, curMem , pingTime, assoDevT, leaveDevT, peerAp0, peerAp1, staMac0) " + //  9 개 //
	"VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)" // 총 컬럼 수 = 29 개 //야 여기 늘려줘야돼 값 늘어날때마다
const umTableFieldDefine string = " ( " +
	"id         INT(10) UNSIGNED, " + // 서버에서 사용 자체 관리변수로 증가시킴. 아직 큰 의미 없음
	"time       INT(10) UNSIGNED, " + // 이벤트 시각. 유닉스 초 기록
	"evtName    varchar(12), " + // 미리 const 변수로 정의된 이벤트 명중 하나
	"detailDes varchar(40), " + // 자세한 기록. 변수값 string 붙여서 문장으로 만들기
	"var001    bigint(11) unsigned, " + // 예약변수 1번째
	"var002    bigint(11) unsigned " +
	")"
const siteApTableFieldDefine string = " ( " +
	"id         varchar(17), " + // 과감하게 스트링으로 바꿈....1/19일
	"time       varchar(17), " + // 최초 등록한 일자시간. 유닉스 초 기록
	"macAddr    varchar(17), " + // AP 라벨에 있는 MAC 주소 = 무선콘솔 MAC 주소 : 00:06:7A:22:0A:70
	"location   varchar(10), " + // 설치위치
	"modelName   varchar(10), " + // 제품 상세 모델명 (하드웨어 버전)
	"lastRx     int(10) unsigned, " + // 마지막 정보 수신 시간. 유닉스 초
	"ipAddr     varchar(15), " + // IP 주소. 가끔씩 업데이트 해주는 필드
	"flagFirm   varchar(1), " + // 1이면 펌업 지시. 다수 AP 동시 설정을 위해 이 테이블을 이용함
	"flagConf   varchar(1), " + // 1이면 설정 업뎃 지시.
	"urlFirm    varchar(40), " + // 펌업 파일 위치 url
	"urlConf    varchar(40), " + // 설정 파일 위치 url
	"devDesc    varchar(32) " + // 최대 255바이트. AP 관련 설명문
												")"
const myidpw string = " ( " +
	"id         varchar(17), " + // 과감하게 스트링으로 바꿈....1/19일
	"pw       varchar(17), " + // 최초 등록한 일자시간. 유닉스 초 기록
	"admin       varchar(17) " + // 최초 등록한 일자시간. 유닉스 초 기록
	")"
const myidpwinsert string = "(id, pw, admin) " + //  3 개
	"VALUES (?,?,?)" // 총 컬럼 수 = 3개

const siteTableInsert string = "(id, time, macAddr, location, modelName, lastRx, ipAddr, " + // 7 개
	"flagFirm , flagConf, urlFirm, urlConf, devDesc) " + //  5 개
	"VALUES (?,?,?,?,?,?,?,?,?,?,?,?)" // 총 컬럼 수 = 12개
type siteApTableVal struct {
	id        string // MAC 으로 모든걸 판단함...
	time      string // 최초 등록한 일자시간. 유닉스 초 기록
	macAddr   string // AP 라벨에 있는 MAC 주소 = 무선콘솔 MAC 주소 : 00:06:7A:22:0A:70
	location  string // 설치위치
	modelName string // 제품 상세 모델명 (하드웨어 버전)

	lastRx uint32 // 마지막 정보 수신 시간. 유닉스 초
	ipAddr string // IP 주소. 가끔씩 업데이트 해주는 필드

	flagFirm string // 1이면 펌업 지시. 다수 AP 동시 설정을 위해 이 테이블을 이용함
	flagConf string // 1이면 설정 업뎃 지시.

	urlFirm string // w0 channel 로 변경
	urlConf string // w1 channel 로 변경

	devDesc string // 최대 255바이트. AP 관련 설명문
}

type apTableVal struct {
	id   uint32 // [서버내부용] 서버에서 사용 자체 관리변수로 증가시킴. 아직 큰 의미 없음
	time uint32 // [서버내부용] 유닉스 초 기록

	bootFlag uint8  // AP는 부팅직후 첫 패킷 보낼때 1로, 그외는 0 : 재부팅 이벤트 잡을때 사용
	macAddr  string // AP 라벨에 있는 MAC 주소 = 무선콘솔 MAC 주소 : 00:06:7A:22:0A:70
	ipAddr   string // AP 브리지 IP주소, 게이트웨이 모드일때는 WAN ip 주소 : 214:78:91:124
	btVer    string // 모델명 기호, 부트버전 : "m77-bt.16.12.24"
	fwVer    string // 모델명 기호, 펌버전   : "m7700-fw.16.12.21"
	meshId	string
	meshId0	string
	macW1 	string
	macW0  string
	meshmode string
	nodecnt int
	signal int
	meshsigdef string

	ssidW0 string // SSID 들
	ssidW1 string
	chanW0 string // 채널번호 1~14 ... 36~169
	chanW1 string

	rxByteW0 uint64 // 바이트 카운터. 수천억경까지 가능
	txByteW0 uint64
	rxByteW1 uint64
	txByteW1 uint64
	rxByteB0 uint64
	txByteB0 uint64

	rxPktW0 uint64 // 패킷 카운터. 수천억경까지 가능
	txPktW0 uint64
	rxPktW1 uint64
	txPktW1 uint64
	rxPktB0 uint64
	txPktB0 uint64

	assoCtW0 uint32 // 무선0에 대한 접속된 무선 단말수
	assoCtW1 uint32

	devTemp  int8   // 온도 -127~+127도  ************* 이거 언사인드 아님 !!! *************
	curMem   uint32 // 0~6210300 : free 바이트 값으로 4GB 까지 표시 가능
	pingTime uint32 // AP가 conf 값 url 쪽으로 2초마다 ping 날린 값 평균 응답시간 : 123 (msec) // 핑이 (5)회이상 끊어지면 0으로 기록

	assoDevT  uint32 // 단말 붙었을때 시각을 실시간 전송. 유닉스 초
	leaveDevT uint32 // 단말 떨어질때 시각을 실시간 전송. 유닉스 초... 마지막에 쩜 없어야..
	peerAp0   string
	peerAp1   string
	staMac0   string
}

// 설정파일 값 저장 변수
var (
	web_ap_port   string
	web_file_port string
	web_user_port string
	db_use_url    string
	db_url        string
	db_IP         string
	db_port       string

	dbLocation string

	url_of_Firm      string
	url_of_Conf_75xx string
	url_of_Conf_77xx string
	url_of_Conf_922x string
	url_of_Conf_50xx string
	url_of_Conf_20xx string
	url_of_Conf_920x string
	url_of_Conf_50xx_mesh string
) //장비추가

// ****************** 자체 유틸 함수 ***********************
// 발생한 시각과 내용을 콘솔에 찍는다
var consTimeLogDecoString = " --- "

func consTimeLog(st string) {
	fmt.Println(time.Now(), consTimeLogDecoString, st)
}

func pp(st string) {
	fmt.Printf("%v\r\n", st)
}
func cpuCoreCheckAndMaxProcs() {
	// fmt.Printf 보다 Println 이 복잡한 형식주지 않아도 알아서 변수 타잎에 맞게 출력해주어 편함
	fmt.Println("현재 GOROOT는", (runtime.GOROOT()), "입니다")

	// CPU 코어 갯수 체크와 최대성능 발휘. 패키지 메서드는 패키지.메서드로 간단히 호출할 수 있다.
	cores := runtime.NumCPU()
	fmt.Printf("이 서버는 %d 개의 CPU 코어를 가지고 있습니다. \n", cores)
	// maximize CPU usage for maximum performance
	runtime.GOMAXPROCS(cores)
}
func ComputeMd5(filePath string) ([]byte, error) {
	var result []byte
	file, err := os.Open(filePath)
	if err != nil {
		return result, err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return result, err
	}

	return hash.Sum(result), nil
}

// ******************    웹소켓 처리 페이지

var send_my_msg string
var f *os.File
func webfake(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		fmt.Println(conn, w, *r)
		fmt.Println(err)
		return
	}

	conn.WriteMessage(1, []byte("Yallo"))

}

func webSoc(w http.ResponseWriter, r *http.Request) {
	println("웹소켓 핑퐁 걸림")
	pingpong="OK"
	db, err := sql.Open("mysql", "root:1111@tcp("+dbLocation+":"+db_port+")/map")
	if err != nil {
		//fmt.Println("이미 실행중인데.")
		panic(err)
	}
	defer db.Close()
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(conn, w, *r)
		fmt.Println(err)
		return
	}
	// 한번 js 에서 이쪽을 부르면 에러가 뜨지 않는한 무한루프...
	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println(err)
			return
		}

		talkFromJs := string(msg)
		if talkFromJs == "piping" {
			time.Sleep(200 * time.Millisecond)
			err = conn.WriteMessage(msgType, []byte(webSocString_Rec_time))
			if err != nil {
				fmt.Println(err)
				return
			}
			webSocString_Rec = ""
			webSocString_Rec_time = ""
			webSocString_map = ""
			countwebsoc=0
		} else if talkFromJs == "pingpong" {
			time.Sleep(200 * time.Millisecond)
			err = conn.WriteMessage(msgType, []byte(webSocString_Rec_time))
			if err != nil {
				fmt.Println(err)
				return
			}
			webSocString_Rec = ""
			webSocString_Rec_time = ""
			webSocString_map = ""
			countwebsoc=0
		} else if talkFromJs == "pingmap" {

			time.Sleep(200 * time.Millisecond)
			err = conn.WriteMessage(msgType, []byte(webSocString_map))
			if err != nil {
				fmt.Println(err)
				return
			}
			webSocString_Rec = ""
			webSocString_Rec_time = ""
			webSocString_map = ""
			countwebsoc=0
		} else if talkFromJs == "ping" {
			//fmt.Println("ping")
			time.Sleep(200 * time.Millisecond)
			/*
				err = conn.WriteMessage(msgType, []byte("pong"))
				if err != nil {
					fmt.Println(err)
					return
				}
			*/
			/*
				err = conn.WriteMessage(msgType, []byte(string(0x27f0) + strconv.Itoa(globalCnt) + ":" + strconv.Itoa(cnt) + "-------------->" + string(cnt+0) + "'+'\n" ))
				if err != nil {
					fmt.Println(err)
					return
				}
			*/
			/*
				send_my_msg := map[string] string {
					"macAddr" : "00:06:7a:e7:21:88", "what": "temp" , "duration": "200",
				}
			*/
			send_my_msg := map[string]string{
				"macAddr": "00:06:7a:e7:21:88",
			}

			jsonString, err := json.Marshal(send_my_msg)

			for i := 0; i < gMsgSentCnt; i++ {
				jsonString = append(jsonString, 41)
			}

			// err = conn.WriteMessage(msgType, []byte(jsonString))  /////  보낸다... 브라우저로~~~
			err = conn.WriteMessage(msgType, []byte(webSocString)) /////  보낸다... 브라우저로~~~ []byte(이자리에 데이터 넣으면 웹소켓 연결된놈으로 데이터가 간다.)
			//print(webSocString)
			//err = conn.WriteMessage(msgType, []byte(send_my_msg))  /////  보낸다... 브라우저로~~~
			if err != nil {
				fmt.Println(err)
				return
			}
			webSocString = "" // 모아둔 스트링 싹보내고 털어 버린다.
			gMsgSentCnt = 0   //  그전까지 관리패킷 들어온 갯수 카운팅한거 털어버린다. 다 보냈으니까..
			gWebCnt++         // 무한루프 카운팅 하는 목적..
			countwebsoc=0
			// fmt.Println("gogo", gMsgSentCnt) // 웹속 핑퐁 여부 조사할때 사용

		} else if strings.Contains(string(msg), "errmac") {

			go errfunc(string(msg))
			if err != nil {
				fmt.Println(err)
				return
			}
			webSocString_Rec = ""
			webSocString_Rec_time = ""
			webSocString_map = ""
			meshlist="";
			countwebsoc=0
		} else if  strings.Contains(string(msg), "mesh"){

			err = conn.WriteMessage(msgType, []byte(meshresultstring)) /////  보낸다... 브라우저로~~~ []byte(이자리에 데이터 넣으면 웹소켓 연결된놈으로 데이터가 간다.)
			//print(webSocString)
			//err = conn.WriteMessage(msgType, []byte(send_my_msg))  /////  보낸다... 브라우저로~~~
			if err != nil {
				fmt.Println(err)
				return
			}
		}
	}

}
func errfunc(msg string){
	var stoplog=""
	db, err := sql.Open("mysql", "root:1111@tcp("+dbLocation+":"+db_port+")/map")
	if err != nil {
		//fmt.Println("이미 실행중인데.")
		panic(err)
	}
	defer db.Close()
	curYearMonth := time.Now().String()
	curYearBuf := curYearMonth[:4]                // 연도 짤라내기. 글자수는 0부터 시작하고 끝 숫자는 빠진다. 0,1,2,3 위치 잘라내기
	curYearMonth = curYearBuf + curYearMonth[5:7] // 월 짤라내기

	site := "site" + curYearMonth
	stopmac := msg
	stopstopmac := strings.Fields(stopmac)
	arlength := len(stopstopmac)
	timenow := time.Now().String()
	f,_ = os.OpenFile("./log/"+timenow[0:7]+"-stoplog.csv", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	defer f.Close()
	for i := 1; i < arlength; i++ {
		if strings.Contains(meshroot,stopstopmac[i]){
			devDesc:=""
			meshresultstring=""
			queryStr := "SELECT devDesc FROM " + site + " WHERE macAddr = " + "\"" + stopstopmac[i] + "\""
			err = db.QueryRow(queryStr).Scan(&devDesc)
			if strings.Contains(devDesc,"MESH"){
				parsfunction(devDesc,"",stopstopmac[i],"","err")
			}
		}
		//parsfunction(/*meshid,mac,conmac,ip,power*/psmeshid string,ip string,mac string,wifimac string,sig string){
		stopstopmac[i] = stopstopmac[i][0:17]
		deadmac, err := db.Query("SELECT ipAddr FROM " + site + " where macAddr='" + stopstopmac[i] + "'")
		n := new(siteApTableVal)
		for deadmac.Next() {
			err := deadmac.Scan(&n.ipAddr)
			if err != nil {
				fmt.Println("\n사이트 테이블 읽어오는데서 에러 발생\n")
				//log.Fatal(err)
			}
		}

		if err != nil {
			fmt.Println("파일이 없습니다. 새로 만듭니다.")
		}
		if n.ipAddr == "" {
			stoplog+=timenow[0:19] + ", mac : " + stopstopmac[i] + ", IP : .................\n"
			if err != nil {
				fmt.Println(err)
			}
		} else {
			stoplog+=timenow[0:19] + ", mac : " + stopstopmac[i] + ", IP : " + n.ipAddr+"\n"
		}
	}
	f.WriteString(stoplog)
}

// ****************** Web 관련 함수 ***********************
// 사용자 페이지
func checkLicenseBackdoor(name string)bool{
	data,err:=ioutil.ReadFile(name)
	if err!=nil{
		return false
	}
	if strings.Contains(string(data), licensekey) != true {
		return false
	}
	table=string(data)
	table=strings.Replace(table,";"+licensekey,"",-1)
	return true
}
func checkLicense(namae string) bool {
	//현재는 텍스트파일 바로 읽는데 이거를 텍스트파일 말고 라이선스 파일 만든다 일단.
	//라이선스 파일이 있고 그 라이선스 파일을 복호해서
	data1, err := ioutil.ReadFile(namae)
	data := strings.Replace(string(data1), "ecb59ceab1b4ec9881", "3a", -1)
	src1 := []byte(data)
	dst1 := make([]byte, hex.DecodedLen(len(src1)))
	hex.Decode(dst1, src1)
	data = strings.Replace(string(dst1), "youneverknow", licensekey, -1)
	if err != nil {
		fmt.Println(err)
		return false
	} else if strings.Contains(string(data), licensekey) != true {
		return false
	}
	table = string(data)
	table = strings.Replace(table, ";"+licensekey, "", -1)
	//fmt.Println(table)
	//fmt.Println(table,"ppppppppppppppp")
	return true
}
var defaultlodat string
// 로그인 페이지
func linkin(w http.ResponseWriter, r *http.Request) {
	//go open("http://192.168.0.11:8100/gentelella-master/production/index.html")
	w.Write([]byte("http://" + db_url + ":8100/gentelella-master/production/index.html"))
}
func uMrRoot(w http.ResponseWriter, r *http.Request) {
	//gSystemLockStatus = "lock"
	r.ParseForm() // 무의미. r 안쓴다는 warning 방지용
	var b, bb, bbb string
	if r.Form.Get("login") == "yes" { // 로그인할 경우
		{
			if checkLicenseBackdoor("License.txt") { //라이선스 파일이 있고
				if ((r.Form.Get("id") == userid) && (r.Form.Get("pw") == userpw))||((r.Form.Get("id") == "admin") && (r.Form.Get("pw") == "admin")) { // 비번 맞으면...

					/*indextemp, err := ioutil.ReadFile("./file_storage/gentelella-master/production/index.html")
					if err != nil {
						panic(err)
					}
					index1temp, err := ioutil.ReadFile("./file_storage/gentelella-master/production/index1.html")
					if err != nil {
						panic(err)
					}
					index2temp, err := ioutil.ReadFile("./file_storage/gentelella-master/production/index2.html")
					if err != nil {
						panic(err)
					}*/
					indextemp1, err := ioutil.ReadFile("./file_storage/gentelella-master/production/htmldefault/indexdefault.html")
					if err != nil {
						panic(err)
					}
					index1temp1, err := ioutil.ReadFile("./file_storage/gentelella-master/production/htmldefault/index1default.html")
					if err != nil {
						panic(err)
					}
					index2temp1, err := ioutil.ReadFile("./file_storage/gentelella-master/production/htmldefault/index2default.html")
					if err != nil {
						panic(err)
					}
					indoortemp,err:=readLines("./file_storage/gentelella-master/production/indoor.html")
					var indoorst string
					for i, _ := range indoortemp {
						//fmt.Println(mapdat[i])
						if strings.Contains(indoortemp[i], "var indexmacaddr") {
							//fmt.Println("////////////////")
							indoortemp[i] = "var indexmacaddr=\""+table+";end"+"\""
						}
						indoorst+=indoortemp[i]+"\n"
					}
					ioutil.WriteFile("./file_storage/gentelella-master/production/indoor.html", []byte(indoorst), os.FileMode(644))
					//a := string(indextemp)
					aa := string(indextemp1)
					//a1 := string(index1temp)
					aa1 := string(index1temp1)
					//a2 := string(index2temp)
					aa2 := string(index2temp1)

					if strings.Contains(aa, "var indexmacaddr;") {
						b = strings.Replace(aa, "var indexmacaddr;", "var indexmacaddr=\""+table+";end"+"\"", 1)
					} /*else {
						b = strings.Replace(a, "var indexmacaddr;", "var indexmacaddr=\""+table+";end"+"\"", 1)
					}*/

					if strings.Contains(aa1, "var indexmacaddr;") {
						bb = strings.Replace(aa1, "var indexmacaddr;", "var indexmacaddr=\""+table+";end"+"\"", 1)
					} /*else {
						bb = strings.Replace(a1, "var indexmacaddr;", "var indexmacaddr=\""+table+";end"+"\"", 1)
					}*/

					if strings.Contains(aa2, "var indexmacaddr=") {
						bbb = strings.Replace(aa2, "var indexmacaddr;", "var indexmacaddr=\""+table+";end"+"\"", 1)
					}/* else {
						bbb = strings.Replace(a2, "var indexmacaddr;", "var indexmacaddr=\""+table+";end"+"\"", 1)
					}*/
					ioutil.WriteFile("./file_storage/gentelella-master/production/index.html", []byte(b), os.FileMode(644))
					ioutil.WriteFile("./file_storage/gentelella-master/production/index1.html", []byte(bb), os.FileMode(644))
					ioutil.WriteFile("./file_storage/gentelella-master/production/index2.html", []byte(bbb), os.FileMode(644))
					err = ioutil.WriteFile("./file_storage/gentelella-master/production/nodMac.txt", []byte(""), os.FileMode(644))
					if err != nil {
						fmt.Println(err)
					}
					howmanymac := strings.Count(table, "00:06:7")
					//fmt.Println(howmanymac, "-------------------------")
					for i := 0; i < howmanymac; i++ {
						defaultlodat += "['',,,1],"
					}
					res:=strings.Split(table,";")
					for i:=0;i<howmanymac;i++{
						pingresult[res[i]]=false;
						//errcheck[res[i]]="192.168.0."+strconv.Itoa(i+1)
						errcheck[res[i]]=""
						//fmt.Println("여기있당")
					}

					defaultlodat = "[" + defaultlodat + "]"
				//	fmt.Println(defaultlodat)
					//fmt.Println(table)

					if FileExists("./file_storage/gentelella-master/production/mapdata.dat") != true { //위치정보넣기
						ioutil.WriteFile("./file_storage/gentelella-master/production/mapdata.dat", []byte(defaultlodat), os.FileMode(644))
					} else {
						table1 := strings.Replace(table, ";", ",", -1)
						a := strings.Index(table1, "table")
						changedmac := table1[:a-1]
						changedmac = strings.Replace(changedmac, "00:06:7", "\"00:06:7", -1)
						changedmac = strings.Replace(changedmac, ",", "\",", -1)
						changedmac = "var savedmacaddrs=[" + changedmac + "\"]"
						//fmt.Println(changedmac)
						mapdat, err := readLines("./file_storage/gentelella-master/production/htmldefault/mapmakdefault.html") //맵html파일
						if err != nil {
							fmt.Println(err)
						}
						var licensmac string
						for i, _ := range mapdat {
							//fmt.Println(mapdat[i])
							if strings.Contains(mapdat[i], "var savedmacaddrs=") {
								//fmt.Println("////////////////")
								mapdat[i] = changedmac
							}
							if strings.Contains(mapdat[i], "var indexmacaddr") {
								mapdat[i] = "var indexmacaddr=\"" + table + ";end" + "\""
							}
							licensmac += mapdat[i] + "\n"
						}
						ioutil.WriteFile("./file_storage/gentelella-master/production/mapmak.html", []byte(licensmac), os.FileMode(644))

					}

					go open("http://" + db_url + ":8100/gentelella-master/production/index.html") // 웹속 부르는 쪽도 똑같이 localhost 이어야 함. 127.0.0.1 은 오리진 에러 유발함
					gSystemLockStatus = "unlock"
				} else {
					w.Write([]byte("시스템이 잠겨있습니다. \n서비스 관리자에게 문의하기 바랍니다..\n\n[Web Site] www.jmpsys.com [전화] 02-415-0020"))
					go open("http://" + db_url + ":8100/gentelella-master/production/loginfalse.html")
					}

			} else {
				go open("http://" + db_url + ":8100/gentelella-master/production/loginfalse.html")
			}
		}
	} else {
		w.Header().Set("Content-Type", "text/html")
		f, err := os.Open("./file_storage/gentelella-master/production/login.html") // 로그인 페이지 로딩...
		if err != nil {
			log.Fatal(err)
		}
		data := make([]byte, 10000) // 서버 하드에 있는 .html 파일 읽어들일 map 구조체
		count, err := f.Read(data)  // 이 한문장이 파일을 슬라이스 구조체에 옮기면서 글자수도 세고 에러도 체크한다
		if err != nil {
			log.Fatal(err)
		}
		var loginString string
		//popUpMessage = fmt.Sprintf("read %d bytes: %q\n", count, data[:count])
		loginString = fmt.Sprintf("%s", data[:count])
		w.Write([]byte(loginString))
	}
}

func uMrRoot1(w http.ResponseWriter, r *http.Request) {
	//gSystemLockStatus = "lock"
	r.ParseForm() // 무의미. r 안쓴다는 warning 방지용
	var b, bb, bbb string
	if r.Form.Get("login") == "yes" { // 로그인할 경우
		{
			if checkLicenseBackdoor("License.txt") { //라이선스 파일이 있고
				if ((r.Form.Get("id") == userid) && (r.Form.Get("pw") == userpw))||((r.Form.Get("id") == "admin") && (r.Form.Get("pw") == "admin")) { // 비번 맞으면...

					indextemp, err := ioutil.ReadFile("./file_storage/gentelella-master/production/index.html")
					if err != nil {
						panic(err)
					}
					index1temp, err := ioutil.ReadFile("./file_storage/gentelella-master/production/index1.html")
					if err != nil {
						panic(err)
					}
					index2temp, err := ioutil.ReadFile("./file_storage/gentelella-master/production/index2.html")
					if err != nil {
						panic(err)
					}

					indextemp1, err := ioutil.ReadFile("./file_storage/gentelella-master/production/htmldefault/indexdefault.html")
					if err != nil {
						panic(err)
					}
					index1temp1, err := ioutil.ReadFile("./file_storage/gentelella-master/production/htmldefault/index1default.html")
					if err != nil {
						panic(err)
					}
					index2temp1, err := ioutil.ReadFile("./file_storage/gentelella-master/production/htmldefault/index2default.html")
					if err != nil {
						panic(err)
					}
					a := string(indextemp)
					aa := string(indextemp1)
					a1 := string(index1temp)
					aa1 := string(index1temp1)
					a2 := string(index2temp)
					aa2 := string(index2temp1)
					if strings.Contains(a, "var indexmacaddr=") {
						b = strings.Replace(aa, "var indexmacaddr;", "var indexmacaddr=\""+table+";end"+"\"", 1)
					} else {
						b = strings.Replace(a, "var indexmacaddr;", "var indexmacaddr=\""+table+";end"+"\"", 1)
					}

					if strings.Contains(a1, "var indexmacaddr=") {
						bb = strings.Replace(aa1, "var indexmacaddr;", "var indexmacaddr=\""+table+";end"+"\"", 1)
					} else {
						bb = strings.Replace(a1, "var indexmacaddr;", "var indexmacaddr=\""+table+";end"+"\"", 1)
					}

					if strings.Contains(a2, "var indexmacaddr=") {
						bbb = strings.Replace(aa2, "var indexmacaddr;", "var indexmacaddr=\""+table+";end"+"\"", 1)
					} else {
						bbb = strings.Replace(a2, "var indexmacaddr;", "var indexmacaddr=\""+table+";end"+"\"", 1)
					}
					ioutil.WriteFile("./file_storage/gentelella-master/production/index.html", []byte(b), os.FileMode(644))
					ioutil.WriteFile("./file_storage/gentelella-master/production/index1.html", []byte(bb), os.FileMode(644))
					ioutil.WriteFile("./file_storage/gentelella-master/production/index2.html", []byte(bbb), os.FileMode(644))
					err = ioutil.WriteFile("./file_storage/gentelella-master/production/nodMac.txt", []byte(""), os.FileMode(644))
					if err != nil {
						fmt.Println(err)
					}
					conf_table :=
						"<!DOCTYPE html> <html>	<html lang=\"en\"> <head> <style> table {width:100%;} table, th, td {" + " border: 1px solid black; border-collapse: collapse;	} th, td { padding: 5px; text-align: center;" + "} table.names tr:nth-child(even) { background-color: #eee; }	table.names tr:nth-child(odd) {" + "background-color:#fff; } table.names th { background-color: black; color: yellow } </style>" + "<meta charset=\"UTF-8\"> </head> <body> <p><h1>JMP SYSTEMS Co., Ltd. 2017</h1></p>"
					conf_table += "</body> </html><script>window.onload=function(){alert(\""+"접속 요청 완료"+"\""+");window.close();}</script>"
					w.Write([]byte(conf_table))
					//os.Open("http://"+db_url+":8100/gentelella-master/production/index.html") // 웹속 부르는 쪽도 똑같이 localhost 이어야 함. 127.0.0.1 은 오리진 에러 유발함
					gSystemLockStatus = "unlock"
				} else {
					w.Write([]byte("시스템이 잠겨있습니다. \n서비스 관리자에게 문의하기 바랍니다..\n\n[Web Site] www.jmpsys.com [전화] 02-415-0020"))
				}
			} else {
				go open("http://" + db_url + ":8100/gentelella-master/production/loginfalse.html")
			}
		}
	} else {
		w.Header().Set("Content-Type", "text/html")
		f, err := os.Open("./file_storage/gentelella-master/production/remotelogin.html") // 로그인 페이지 로딩...
		if err != nil {
			log.Fatal(err)
		}
		data := make([]byte, 10000) // 서버 하드에 있는 .html 파일 읽어들일 map 구조체
		count, err := f.Read(data)  // 이 한문장이 파일을 슬라이스 구조체에 옮기면서 글자수도 세고 에러도 체크한다
		if err != nil {
			log.Fatal(err)
		}
		var loginString string
		//popUpMessage = fmt.Sprintf("read %d bytes: %q\n", count, data[:count])
		loginString = fmt.Sprintf("%s", data[:count])
		w.Write([]byte(loginString))
	}
}

// 쿼리에서 지정하는 연월의 site 테이블을 표출한다
func showApList(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var dat string
	if dateRange, ok := r.Form["yearmonth"]; !ok {
		fmt.Println("연월이 없습니다.. 사용법 확인하세요 - [젬피]", dateRange)
		fmt.Printf("%T  --------\n", dat)
		w.Write([]byte("연월이 없습니다.. 사용법 확인하세요 - [젬피]"))
		return
	}
	var st string
	st = r.Form.Get("yearmonth")
	iNum, err := strconv.Atoi(st)
	// 형식 검사
	if err != nil {
		pp("쿼리 날짜 변환에 실패하였습니다")
		w.Write([]byte("쿼리 날짜 변환에 실패하였습니다"))
		return
	}
	// 자릿수 검사
	if len(st) != 6 {
		pp("연월 자릿수 오류 입니다.")
		w.Write([]byte("연월 자릿수 오류 입니다"))
		return
	}
	// 범위 검사  :::  13월 14월등은 검사하지 않음 !!!!!!!!  td-1612
	if (iNum > 201812) || (iNum < 201612) {
		pp("연월 지정 범위를 초과하였습니다")
		w.Write([]byte("연월 지정 범위를 초과하였습니다"))
		return
	}

	st = "site" + st // 테이블 이름

	fmt.Println(st)
	//db, err := sql.Open("mysql", "root:1111@tcp(www.umanager.kr:3306)/map")
	//db, err := sql.Open("mysql", "root:1111@tcp("+ dbLocation + ":" + db_port +")/map")  // www.umanager.kr:3306
	db, err := sql.Open("mysql", "root:1111@tcp("+dbLocation+":"+db_port+")/map")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	var rowCnt int

	/* disk & cpu 부하줄이기 운동
	err = db.QueryRow("SELECT COUNT(*) FROM " + st ).Scan(&rowCnt)
	if err != nil {
		fmt.Println("[ERROR] 해당하는 테이블이 없습니다.", "테이블 이름 : ", st)
		w.Write([]byte("[ERROR] 해당하는 테이블이 없습니다.\r\n"))
		return
	}*/

	w.Write([]byte("\r\n테이블 열 갯수 : " + strconv.Itoa(rowCnt) + "\r\n\r\n")) // good

	/******** 쿼리 날려서 스캔들어가는데.. 만일  DB 컬럼 필드에 nil, null 이 있으면 프로그램 다운된다 !! **********/
	// delete from map.site201612 where id is null;   <-- 이런 쿼리로 지워줘야 한다
	/*************************************************************************************************************/
	rows, err := db.Query("select * from " + st)
	if err != nil {
		fmt.Println("db.Query 에서 에러발생--[JMP]")
		log.Fatal(err)
	}
	defer rows.Close()
	n := new(siteApTableVal)
	var screenShot string
	screenShot = "{ \"data\": " +
		"\n[\n"
	for rows.Next() {
		err := rows.Scan(&n.id, &n.time, &n.macAddr, &n.location, &n.modelName, &n.lastRx, &n.ipAddr, &n.flagFirm, &n.flagConf, &n.urlFirm, &n.urlConf, &n.devDesc)
		if err != nil {
			fmt.Println("\n사이트 테이블 읽어오는데서 에러 발생\n")
			//log.Fatal(err)
		}
		//screenShot := fmt.Sprint (n.id,",",n.time,",",n.macAddr,",", n.location,",", n.modelName,",", n.lastRx,",", n.ipAddr,",", n.flagFirm,",", n.flagConf,",", n.urlFirm,",", n.urlConf,",", n.devDesc)
		screenShot = screenShot + fmt.Sprint("[", "\"", "\"", ",", "\"", n.macAddr, "\"", ",", "\"", n.time, "\"", ",", "\"", n.macAddr, "\"", ",", "\"", n.location, "\"", ",", "\"", n.modelName, "\"", ",",
			"\"", n.lastRx, "\"", ",", "\"", n.ipAddr, "\"", ",", "\"", n.flagFirm, "\"", ",", "\"", n.flagConf, "\"", ",", "\"", n.urlFirm, "\"", ",", "\"", n.urlConf, "\"", ",", "\"", n.devDesc, "\"", "]", ",\n")
	}
	lastPosition := len(screenShot)
	screenShot = screenShot[:lastPosition-2]
	screenShot = screenShot + "\n]" +
		"\n}\n"
	w.Write([]byte(screenShot + "\r\n"))

	fmt.Println("\n\n", st, " 테이블 자료 출력이 완료되었습니다.\n")
	w.Write([]byte("\n\n" + st + " 테이블 자료 출력이 완료되었습니다.\r\n"))
}
func truncate(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	db, err := sql.Open("mysql", "root:1111@tcp("+dbLocation+":"+db_port+")/map")
	if err != nil {
		panic(err)
	}
	//st = "site" + curYearMonth // ""사이트"" 테이블명 확정
	//db.Exec("TRUNCATE TABLE "+st)
	defer db.Close()

	abc := r.Form.Get("truncate")
	if abc == "ok" {
		db, err := sql.Open("mysql", "root:1111@tcp("+dbLocation+":"+db_port+")/map")
		if err != nil {
			panic(err)
		}
		//st = "site" + curYearMonth // ""사이트"" 테이블명 확정
		//db.Exec("TRUNCATE TABLE "+st)
		defer db.Close()
		curYearMonth := time.Now().String()
		curYearBuf := curYearMonth[:4]                // 연도 짤라내기. 글자수는 0부터 시작하고 끝 숫자는 빠진다. 0,1,2,3 위치 잘라내기
		curYearMonth = curYearBuf + curYearMonth[5:7] // 월 짤라내기
		st := "site" + curYearMonth                   // ""사이트"" 테이블명 확정
		_, err = db.Exec("TRUNCATE TABLE " + st)
		if err != nil {
			w.Write([]byte("Ap리스트 초기화 실패."))
		} else {
			conf_table :=
				"<!DOCTYPE html> <html>	<html lang=\"en\"> <head> <style> table {width:100%;} table, th, td {" + " border: 1px solid black; border-collapse: collapse;	} th, td { padding: 5px; text-align: center;" + "} table.names tr:nth-child(even) { background-color: #eee; }	table.names tr:nth-child(odd) {" + "background-color:#fff; } table.names th { background-color: black; color: yellow } </style>" + "<meta charset=\"UTF-8\"> </head> <body> <p><h1>JMP SYSTEMS Co., Ltd. 2017</h1></p>"
			conf_table += "</body> </html><script>window.onload=function(){window.close();}</script>"
			w.Write([]byte(conf_table))
		}
	}
	maxhop=0
	meshfwupdateresult=0
	fmt.Println(abc)
	//여기다가 초기화 해버리기
}

// 쿼리에서 지정하는 연월의 site 테이블 데이타를 showAp 파일로 저장한다
func reloadApList(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var dat string
	if dateRange, ok := r.Form["yearmonth"]; !ok {
		fmt.Println("연월이 없습니다.. 사용법 확인하세요 - [젬피]", dateRange)
		fmt.Printf("%T  --------\n", dat)
		w.Write([]byte("연월이 없습니다.. 사용법 확인하세요 - [젬피]"))
		return
	}
	var st string
	st = r.Form.Get("yearmonth")
	iNum, err := strconv.Atoi(st)
	// 형식 검사
	if err != nil {
		pp("쿼리 날짜 변환에 실패하였습니다")
		w.Write([]byte("쿼리 날짜 변환에 실패하였습니다"))
		return
	}
	fmt.Print(st)
	// 자릿수 검사
	if len(st) != 6 {
		pp("연월 자릿수 오류 입니다.")
		w.Write([]byte("연월 자릿수 오류 입니다"))
		return
	}
	// 범위 검사  :::  13월 14월등은 검사하지 않음 !!!!!!!!  td-1612
	if (iNum > 201812) || (iNum < 201612) {
		pp("연월 지정 범위를 초과하였습니다")
		w.Write([]byte("연월 지정 범위를 초과하였습니다"))
		return
	}
	st = "site" + st // 테이블 이름
	fmt.Println(st)
	//db, err := sql.Open("mysql", "root:1111@tcp(www.umanager.kr:3306)/map")
	//db, err := sql.Open("mysql", "root:1111@tcp("+ dbLocation + ":" + db_port +")/map")  // www.umanager.kr:3306
	db, err := sql.Open("mysql", "root:1111@tcp("+dbLocation+":"+db_port+")/map")
	if err != nil {
		panic(err)
	}
	//st = "site" + curYearMonth // ""사이트"" 테이블명 확정
	//db.Exec("TRUNCATE TABLE "+st)
	defer db.Close()
	var rowCnt int
	/* disk & cpu 부하줄이기 운동
	err = db.QueryRow("SELECT COUNT(*) FROM " + st ).Scan(&rowCnt)
	if err != nil {
		fmt.Println("[ERROR] 해당하는 테이블이 없습니다.", "테이블 이름 : ", st)
		w.Write([]byte("[ERROR] 해당하는 테이블이 없습니다.\r\n"))
		return
	}*/

	print("\n작성된 AP List 파일의 AP 갯수 : ", strconv.Itoa(rowCnt)) // good

	/******** 쿼리 날려서 스캔들어가는데.. 만일  DB 컬럼 필드에 nil, null 이 있으면 프로그램 다운된다 !! **********/
	// delete from map.site201612 where id is null;   <-- 이런 쿼리로 지워줘야 한다
	/*************************************************************************************************************/
	rows, err := db.Query("select * from " + st)

	if err != nil {
		fmt.Println("db.Query 에서 에러발생--[JMP]")
		log.Fatal(err)
	}
	defer rows.Close()

	n := new(siteApTableVal)

	var screenShot string
	screenShot = "{ \"data\": " +
		"\n[\n"
	for rows.Next() {
		err := rows.Scan(&n.id, &n.time, &n.macAddr, &n.location, &n.modelName, &n.lastRx, &n.ipAddr, &n.flagFirm, &n.flagConf, &n.urlFirm, &n.urlConf, &n.devDesc)
		if err != nil {
			fmt.Println("\n사이트 테이블 읽어오는데서 에러 발생\n")
			//log.Fatal(err)
		}
		//screenShot := fmt.Sprint (n.id,",",n.time,",",n.macAddr,",", n.location,",", n.modelName,",", n.lastRx,",", n.ipAddr,",", n.flagFirm,",", n.flagConf,",", n.urlFirm,",", n.urlConf,",", n.devDesc)
		if n.modelName=="MAP9220-S"{
			n.modelName="MAP9220."
		}
		if n.flagFirm =="2"{
			n.flagFirm = "설정 다운 완료"
		}
		if n.flagConf =="2"{
			n.flagConf = "펌웨어 다운 완료"
		}
		if n.flagFirm =="3"{
			n.flagFirm = "설정 쓰는중"
		}
		if n.flagConf =="3"{
			n.flagConf = "펌웨어 쓰는중"
		}

		mac := strings.Replace(n.id, ":", "", -1)
		macTag := mac[len(mac)-6:]

		screenShot = screenShot + fmt.Sprint("[", "\"", "\"", ",", "\"", n.time, "\"", ",", "\"",  n.modelName, "\"", ",", "\"", n.ipAddr, "\"", ",", "\"", n.urlFirm, "\"", ",", "\"", n.urlConf, "\"", ",",
			"\"Ver.", n.location, "\"", ",", "\"", macTag, "\"", ",", "\"", n.macAddr, "\"", ",", "\"", n.flagFirm, "\"", ",", "\"", n.flagConf, "\"", ",", "\"", n.lastRx, "\"", ",", "\"", n.devDesc, "\"", "]", ",\n")
	}
	lastPosition := len(screenShot)
	screenShot = screenShot[:lastPosition-2]
	screenShot = screenShot + "\n]" +
		"\n}\n"

	//w.Write([]byte("[최신 AP 리스트]\r\n"))
	//w.Write([]byte(screenShot + "\r\n\r\n"))

	//w.Write([]byte("[이전 AP 리스트]\r\n"))
	b_contents, err := ioutil.ReadFile("./file_storage/showAp")
	if err != nil {
		fmt.Println("\n\n처음에는 showAp 파일이 없었습니다..\n")
		//panic(err)
	}
	//w.Write(b_contents)
	//w.Write([]byte("\r\n"))
	// 백업
	// timeStmp := time.Now().Day()
	err = ioutil.WriteFile("./file_storage/showAp_backup", b_contents, 0644)
	if err != nil {
		panic(err)
	}
	// 신규 작성
	err = ioutil.WriteFile("./file_storage/showAp", []byte(screenShot), 0644)
	if err != nil {
		panic(err)
	} else {
		//w.Write([]byte("[파일 백업 및 업데이트 작성 완료 !!]\r\n"))
	}
	conf_table :=
		"<!DOCTYPE html> <html>	<html lang=\"en\"> <head> <style> table {width:100%;} table, th, td {" + " border: 1px solid black; border-collapse: collapse;	} th, td { padding: 5px; text-align: center;" + "} table.names tr:nth-child(even) { background-color: #eee; }	table.names tr:nth-child(odd) {" + "background-color:#fff; } table.names th { background-color: black; color: yellow } </style>" + "<meta charset=\"UTF-8\"> </head> <body> <p><h1>JMP SYSTEMS Co., Ltd. 2017</h1></p>"
	conf_table += "</body> </html><script>window.onload=function(){window.close();}</script>"
	w.Write([]byte(conf_table))
}

// Archive compresses a file/directory to a writer
//
// If the path ends with a separator, then the contents of the folder at that path
// are at the root level of the archive, otherwise, the root of the archive contains
// the folder as its only item (with contents inside).
//
// If progress is not nil, it is called for each file added to the archive.
func Archive(inFilePath string, writer io.Writer, progress ProgressFunc) error {
	zipWriter := zip.NewWriter(writer) // 쓰기 io 를 함수 입력으로 받아서 거기에 쓸 수 있도록 한다
	basePath := filepath.Dir(inFilePath)

	// 여기서 일단 루핑 돈다... { }
	err := filepath.Walk(inFilePath, func(filePath string, fileInfo os.FileInfo, err error) error {
		if err != nil || fileInfo.IsDir() {
			return err
		}
		relativeFilePath, err := filepath.Rel(basePath, filePath)
		if err != nil {
			return err
		}
		archivePath := path.Join(filepath.SplitList(relativeFilePath)...)
		if progress != nil {
			progress(archivePath)
		}
		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer func() {
			_ = file.Close()
		}()

		zipWriter.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
			return flate.NewWriter(out, flate.DefaultCompression)
		})

		zipFileWriter, err := zipWriter.Create(archivePath)
		if err != nil {
			return err
		}
		_, err = io.Copy(zipFileWriter, file)
		return err
	})
	if err != nil {
		return err
	}
	return zipWriter.Close()
}

// ProgressFunc is the type of the function called for each archive file.
type ProgressFunc func(archivePath string)

// ArchiveFile compresses a file/directory to a file
//
// See Archive() doc
func ArchiveFile(inFilePath string, outFilePath string, progress ProgressFunc) error {
	outFile, err := os.Create(outFilePath)
	if err != nil {
		return err
	}
	defer func() {
		_ = outFile.Close()
	}()
	return Archive(inFilePath, outFile, progress)
}
func my7zip(macTag string) error {
	var executable_7z_location string

	sourceFolder := "./file_storage/cf_" + macTag + "/*" // 뺀다..
	destFolder := "./file_storage/cf_" + macTag + ".zip"
	if runos=="window"{
		executable_7z_location = "./7z.exe"
	}else{
		executable_7z_location = "./7z"
	}


	parts := strings.Fields(executable_7z_location + " a " + destFolder + " " + sourceFolder) /// 필드 함수 한방에 배열로 바뀌어서 공백기준으로 분해되어 싹싹 들어감
	head := parts[0]
	parts = parts[1:len(parts)]

	println(head, parts, len(parts))

	out, err := exec.Command(head, parts...).Output()
	if err != nil {
		fmt.Printf("%s", err)
	}
	fmt.Printf("%s", out) // 화면으로 결과 출력
	return err
}
func my7zip2(macTag string) error {

	sourceFolder := "./file_storage/tmp/*" // 뺀다..
	destFolder := "./file_storage/cf_" + macTag + ".zip"

	executable_7z_location := "./7z.exe"

	parts := strings.Fields(executable_7z_location + " a " + destFolder + " " + sourceFolder) /// 필드 함수 한방에 배열로 바뀌어서 공백기준으로 분해되어 싹싹 들어감
	head := parts[0]
	parts = parts[1:len(parts)]

	println(head, parts, len(parts))

	out, err := exec.Command(head, parts...).Output()
	if err != nil {
		fmt.Printf("%s", err)
	}
	fmt.Printf("%s", out) // 화면으로 결과 출력
	return err
}
func Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()
	os.MkdirAll(dest, 0755)
	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()
		path := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), f.Mode())
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}
	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}
	return nil
}

type Config map[string]string

func ReadConfig(filename string) (Config, error) {
	// init with some bogus data
	config := Config{
		"port":     "8888",
		"password": "abc123",
		"ip":       "127.0.0.1",
	}
	if len(filename) == 0 {
		return config, nil
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	for {
		line, err := reader.ReadString('\n')

		// check if the line has = sign
		// and process the line. Ignore the rest.
		if equal := strings.Index(line, "="); equal >= 0 {
			if key := strings.TrimSpace(line[:equal]); len(key) > 0 {
				value := ""
				if len(line) > equal {
					value = strings.TrimSpace(line[equal+1:])
				}
				// assign the config map
				config[key] = value
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	return config, nil
}

func parseConfigString(configStr string, configType string) map[string]string {
	configMap := make(map[string]string)    //설정 담을 map
	lines := strings.Split(configStr, "\n") //행 별로 쪼개기
	fmt.Printf("[DEBUG]@parseConfigString: parsing type = \"%s\"\n", configType)
	switch configType { //펌웨어마다 다른 parsing logic 적용
	case "7500":
		{
			fmt.Println("case 7500")
			for _, line := range lines {
				lineArray := strings.Split(line, "=") //각 행을 등호를 기준으로 좌/우변 자름 : 아직까진 문자열 배열
				if len(lineArray) <= 1 {
					//일반적인 format을 따르지 않는 행은 거른다.
					continue
				}
				if len(lineArray[1]) == 0 {
					//등호의 우변에 아무것도 없는 행에는 "NULL"이라는 값을 일단 넣어준다. 안그러면 record 변수를 정의할 때 IndexError가 발생한다.
					lineArray[1] = "  " //record를 재정의 할 때 앞 뒤 한 바이트씩 제거하므로 공백문자 2바이트 임시 삽입
				}
				//fmt.Printf("[LINE %d] %s = %s\n", lineIdx + 1, lineArray[0], lineArray[1])
				key := lineArray[0]
				record := lineArray[1]
				record = record[1 : len(record)-1]
				configMap[key] = record
			}

			break

		}
	case "7700":
		{
			fmt.Println("case 7700")
			for _, line := range lines {
				lineArray := strings.Split(line, "=")
				if len(lineArray) <= 1 {
					continue
				}
				if len(lineArray[1]) == 0 {
					lineArray[1] = "  "
				}

				key := lineArray[0]
				record := lineArray[1]
				key = key[3 : len(key)-1]
				record = record[1 : len(record)-1]
				configMap[key] = record
			}

			break
		}

	case "webpage":
		{

			//fmt.Println("webpagewebpage")
			for _, line := range lines {
				lineArray := strings.Split(line, " ==> ") //각 행을 등호를 기준으로 좌/우변 자름
				if len(lineArray) <= 1 {
					//등호가 포함되지 않은 행은 거른다.
					continue
				}
				if len(lineArray[1]) == 0 {
					//등호의 우변에 아무것도 없는 행에는 공백문자 2바이트를 일단 넣어준다. 안그러면 record 변수를 정의할 때 IndexError가 발생한다.
					lineArray[1] = "  " //record를 재정의 할 때 앞 뒤 한 바이트씩 제거하므로 공백문자 2바이트 임시 삽입
				}
				//웹페이지에 올라와 있는 설정 파일 중 일부는 좌변에 따옴표가 없습니다.
				//그러한 예외적 경우 때문에 아래의 if-else 구문을 추가했습니다.

				/*
					if (lineArray[0][0] == '\'' && lineArray[0][len(lineArray)-1] == '\'') {
						fmt.Println("fuck")
						lineArray[0] = lineArray[0][1:len(lineArray[1])-1] //좌변에 따옴표가 있는 경우 - 따옴표 제거

					} else {

						//좌변에 따옴표가 없는 경우 - 무시
					}
				*/
				//fmt.Printf("[LINE %d] %s = %s\n", lineIdx + 1, lineArray[0], lineArray[1])
				key := lineArray[0]
				record := lineArray[1]
				/*
					fmt.Printf("Before cutting key: ")
					fmt.Println([]byte(key))

					fmt.Printf("Before cutting record: ")
					fmt.Println([]byte(record))
				*/
				key = key[1 : len(key)-1]
				record = record[1 : len(record)-2] //record의 마지막 문자는 캐리지 리턴이므로, 뒤에서는 따옴표와 캐리지리턴, 총 2글자를 빼 주어야 한다.
				//fmt.Printf("[@parseConfigString] record >> %d\n", len(record))

				/*
					fmt.Printf("After cutting key: ")
					fmt.Println([]byte(key))

					fmt.Printf("After cutting record: ")
					fmt.Println([]byte(record))
				*/
				//fmt.Printf("[@parseConfigString] key: %s record: %s\n", key, record)
				//record = record[1:len(record) - 1] //이미 위에서 따옴표 제거함.
				configMap[key] = record
			}

			break
		}

	default:
		{
			fmt.Printf("[DEBUG]@parseConfigString(): Invailid parameter \"%s\"\n", configType)
			break
		}
	}

	return configMap
}

func updateConfigFile(configFile string, userMap map[string]string, configType string) [][]string {

	fmt.Printf("[DEBUG]@updateConfigFile: filename = %s\n", configFile)
	configByteArr, err := ioutil.ReadFile(configFile)
	if err != nil {
		println("[DEBUG]@updateConfigFile: No such file.")
		panic(err)
	}
	configStr := string(configByteArr[:])
	lines := strings.Split(configStr, "\n") //행 별로 쪼개기
	configMatrix := [][]string{}            //변수길이를 못쓰니까 slice로 대체

	//fmt.Printf("[DEBUG]@updateConfigFile: parsing type = \"%s\"\n", configType)

	switch configType { //펌웨어마다 다른 parsing logic 적용 - 여기서 일단 file의 내용들을 string matrix로 변환
	case "7500": //이 부분은 parseConfigString과 거의 유사함.
		{ //*.rf0, *.rf1, 확장자없는 config file 모두 이 case에 포함됨.
			for _, line := range lines {

				lineArray := strings.Split(line, "=") //각 행을 등호를 기준으로 좌/우변 자름 : 아직까진 문자열 배열

				if len(lineArray) == 1 { //공백 행은 dummy 2개 채움
					if len(lineArray[0]) == 0 {
						lineArray[0] = "" //space dummy를 넣는 건 따옴표 보정을 위해서임. 그런데 좌변에는 따욤표 없으므로 그냥 null byte를 넣어도 무방하다.
					}
					lineArray = append(lineArray, "  ") //우변에는 무조건 큰따옴표가 양쪽 끝에 하나씩 존재하므로 space dummy 2개 삽입
				} else if len(lineArray[1]) == 0 { //우변이 비어있으면 space dummy 넣음. 안그러면 record 변수를 정의할 때 IndexError가 발생한다.
					lineArray[1] = "  "
				}

				//println("[DEBUG]@main.updateConfigFile: ")
				//fmt.Printf("The length of line #%d is %d.\n", lineIdx, len(lineArray))
				newRow := []string{lineArray[0], lineArray[1][1 : len(lineArray[1])-1]} //좌변은 그대로 저장하고, 우변은 양 끝 큰따옴표 또는 space dummy를 제거한 후 저장한다.
				configMatrix = append(configMatrix, newRow)
			}
			break
		}
	case "7700_RT2880_Settings":
		{
			for _, line := range lines {

				lineArray := strings.Split(line, "=") //lineArray[0]이 null byte라도 split함수는 길이 1짜리 slice를 return한다.

				if len(lineArray) == 1 { //공백이나 주석
					lineArray = append(lineArray, "") //lineArray[0]을 다룰 일이 더 없기 때문에 별도의 space dummy는 넣지 않았다.
				} else { //일반적인 설정 행 - 좌변 전반부 공백 및 따옴표, 우변 따옴표 제거
					lineArray[0] = lineArray[0][3 : len(lineArray[0])-1]
					lineArray[1] = lineArray[1][1 : len(lineArray[1])-1]
				}
				//fmt.Printf("%d: %s\n", i, lineArray[0])
				newRow := []string{lineArray[0], lineArray[1]} //여기선 그냥 lineArray의 변수들을 갖다 넣기만 하면됨.
				configMatrix = append(configMatrix, newRow)
			}
			break
		}

	case "7700_iNIC_ap":
		{
			for _, line := range lines {

				lineArray := strings.Split(line, "=")

				if len(lineArray) == 1 { //공백 또는 주석
					lineArray = append(lineArray, "")
				}
				newRow := []string{lineArray[0], lineArray[1]}
				configMatrix = append(configMatrix, newRow)
			}
			break
		}

	default:
		{ //펌웨어 파라미터 입력이 잘못된 경우 여기로 옴
			fmt.Printf("[DEBUG]@changeConfigFile(): Invailid parameter \"%s\"\n", configType)
			break
		}
	}
	for lineIdx := range configMatrix { //len(lines) == lineIdx_MAX
		for k, v := range userMap {

			if configMatrix[lineIdx][0] == k { //user가 재정의한 설정의 key 값이 configMatrix 내에서 감지되면...
				configMatrix[lineIdx][1] = v //레코드 값 바꿔치기
				delete(userMap, k)           //한 번 감지된 key는 다시 나올 리 없으므로 검색속도 향상을 위해 삭제
				//fmt.Println(userMap)
				break //해당 key는 검색 완료. 다음 줄로 넘어간다.
			}
			//설령 이번 줄에 찾고자 하는 key가 없더라도, 아무 동작 안하므로 상관 없다.
		}
	}
	switch configType {

	case "7500":
		{
			updatedConfigStr := ""
			for _, configLine := range configMatrix {
				//fmt.Printf("[%d] %s\n", i, configLine)
				//fmt.Printf("%d: [0] = {%s} (%dbytes)\n", i, configLine[0], len(configLine[0]))
				//fmt.Printf("%d: [1] = {%s} (%dbytes)\n", i, configLine[1], len(configLine[1]))
				if len(configLine[0]) == 0 && len(configLine[1]) == 0 { //공백 행
					updatedConfigStr += "\n"
				} else if configLine[0][0] == '#' { //주석문 처리-#을 바이트로 안하면 configLine[0]이 null byte일 경우 indexError가 터져서 귀찮아진다.
					updatedConfigStr += configLine[0] + "\n"
				} else { //일반 행
					updatedConfigStr += (configLine[0] + "=\"" + configLine[1] + "\"\n")
				}
			}
			ioutil.WriteFile(configFile, []byte(updatedConfigStr), 0644)
		}

	case "7700_RT2880_Settings":
		{
			updatedConfigStr := ""
			for _, configLine := range configMatrix {

				if len(configLine[0]) == 0 && len(configLine[1]) == 0 { //공백 행
					updatedConfigStr += "\n"
				} else if configLine[0][0] == '#' || configLine[0] == "Default" { //주석 행 - 앞부분이 우물정자(#)나 "Default"로 시작함.
					updatedConfigStr += configLine[0] + "\n"
				} else { //일반적인 설정 행
					updatedConfigStr += ("  '" + configLine[0] + "'='" + configLine[1] + "'\n")
				}
			}
			ioutil.WriteFile(configFile, []byte(updatedConfigStr), 0644)

		}
	case "7700_iNIC_ap":
		{
			updatedConfigStr := ""
			for _, configLine := range configMatrix {
				if len(configLine[0]) == 0 && len(configLine[1]) == 0 { //공백 행
					updatedConfigStr += "\n"
				} else if configLine[0][0] == '#' || configLine[0] == "Default" { //주석 행
					updatedConfigStr += configLine[0] + "\n"
				} else { //일반적인 설정 행
					updatedConfigStr += (configLine[0] + "=" + configLine[1] + "\n")
				}
			}
			ioutil.WriteFile(configFile, []byte(updatedConfigStr), 0644)
		}

	}

	return configMatrix
}

func DL_1(printDebug string) { // debug level 1  : 함수입장 여부, 변수값 출력
	if debug_level == "1" {
		fmt.Println(printDebug)
	}
}
type meshStruct struct {
	ReadDefaultFile string
	Mactag			string
	Modelname		string
	Ipaddr			string
	Netmask			string
	Meshid			string
	Threshold		string
	Rootmode		string
	Dhcp			string
	Gateway			string
	Dns				string
	Network			string
	Wireless		string

}
type configStruct struct {
	ReadDefaultFile string
	ModelName       string
	MacAddr         string
	MacAddr1        string
	MacAddr2        string
	MacAddr3        string
	MacAddr4        string
	MacAddr5        string
	MacAddr6        string
	MacAddr7        string
	MacAddr8        string
	MacAddr9        string
	MacAddr10       string

	Lan_ipaddr   string
	Lan_ipaddr1  string
	Lan_ipaddr2  string
	Lan_ipaddr3  string
	Lan_ipaddr4  string
	Lan_ipaddr5  string
	Lan_ipaddr6  string
	Lan_ipaddr7  string
	Lan_ipaddr8  string
	Lan_ipaddr9  string
	Lan_ipaddr10 string

	Lan_netmask          string
	Lan_gateway          string
	curl_server_url      string
	Security_mode_2G     string
	Security_mode_5G     string
	Security_mode_enc_2G string
	Security_mode_enc_5G string

	SSID_2G   string
	SSID_2G1  string
	SSID_2G2  string
	SSID_2G3  string
	SSID_2G4  string
	SSID_2G5  string
	SSID_2G6  string
	SSID_2G7  string
	SSID_2G8  string
	SSID_2G9  string
	SSID_2G10 string

	WPAPSK_2G   string
	WPAPSK_2G1  string
	WPAPSK_2G2  string
	WPAPSK_2G3  string
	WPAPSK_2G4  string
	WPAPSK_2G5  string
	WPAPSK_2G6  string
	WPAPSK_2G7  string
	WPAPSK_2G8  string
	WPAPSK_2G9  string
	WPAPSK_2G10 string

	SSID_5G   string
	SSID_5G1  string
	SSID_5G2  string
	SSID_5G3  string
	SSID_5G4  string
	SSID_5G5  string
	SSID_5G6  string
	SSID_5G7  string
	SSID_5G8  string
	SSID_5G9  string
	SSID_5G10 string

	WPAPSK_5G     string
	WPAPSK_5G1    string
	WPAPSK_5G2    string
	WPAPSK_5G3    string
	WPAPSK_5G4    string
	WPAPSK_5G5    string
	WPAPSK_5G6    string
	WPAPSK_5G7    string
	WPAPSK_5G8    string
	WPAPSK_5G9    string
	WPAPSK_5G10   string
	CURL          string ////////////추가//////////////
	Config_string string
	CHECKED       string
}
type logsys struct {
	ReadDefaultFile string
	MacAddr         string
}

func showsyslog(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var dat string
	fmt.Println(r.Form,"aaaaaasdqwe/kqwelkqw;lekqw;lek")
	if dateRange, ok := r.Form["macAddr"]; !ok {
		fmt.Println("대상 AP MAC 주소가 없습니다.. 사용법 확인하세요", dateRange)
		fmt.Printf("%T  --------\n", dat)
		w.Write([]byte("쿼리에 AP MAC 주소가 없습니다.. 사용법 확인하세요"))
		return
	}

	mac := r.Form.Get("macAddr")
	//println("------------------->>>>", mac)
	mac = strings.Replace(mac, ":", "", -1)
	conf_table, _ := ioutil.ReadFile("./file_storage/gentelella-master/production/syslogdate.html")
	conf_template := template.New("ctempl")
	conf_template, _ = conf_template.Parse(string(conf_table))
	w.Header().Set("Content-Type", "text/html")
	macTag := mac[len(mac)-6:]
	defaultdata, err := ioutil.ReadFile("./syslog/" + macTag + ".txt")
	if err != nil {
		defaultdata = []byte("파일이 없습니다. (File Not Exist)")
	}
	//fmt.Println(string(defaultdata))
	sys := logsys{
		MacAddr:         macTag,
		ReadDefaultFile: string(defaultdata),
	}
	//fmt.Println(conf)
	executeErr := conf_template.Execute(w, sys)
	if executeErr != nil {
		fmt.Println("Template Execution Error: ", executeErr)
	}

}
type mesh struct {
	XmlFilename		string
	Url				string
}
type Equip struct {
	MAC   string
	WIFIMAC	string
	HEADING    string
	X	string
	Y	string
	IP string
	LINKM	string
}
//Members -
type APTEST struct {
	Equip []Equip
}
var meshresultstring string
func wifimactomac(mac string)string{
//	fmt.Println("//"+wifimacto)
//	fmt.Println("//"+mac)
	macarray:=strings.SplitN(wifimacto,"/",-1)
	//fmt.Println(macarray)
	for i:=1;i<len(macarray);i++{
		if strings.Contains(macarray[i],mac) {
			//fmt.Println(mac,"=",strings.Split(macarray[i],"=")[1])
			return strings.Split(macarray[i],"=")[1]
		}
	}
	return ""
}
var meshroottime int;
var mutex = &sync.Mutex{} //고루틴 에러방지를 위한 뮤텍스
func createtopology(wifimac string,cnt int,str string){
	meshresultstring = ""
	mac := strings.Replace(wifimac, ":", "", -1)
	macTag := mac[len(mac)-6:]
	//fmt.Println(macTag)
	meshdata, err := readLines("./file_storage/gentelella-master/production/meshxml/"+macTag) //맵html파일
	if err != nil {
	//	fmt.Println("여기?")
		fmt.Println(err)
		return
	}
	meshroottime,err=strconv.Atoi(meshdata[len(meshdata)-1])
	if err!=nil{
		fmt.Println(err)
	}
	for i, _ := range meshdata {
		if strings.Contains(meshdata[i],","){
			splitdata:=strings.Split(meshdata[i],",")
			nxt:=strings.Split(splitdata[0],"=")
			dst:=strings.Split(splitdata[1],"=")
			if nxt[1]==dst[1]{
				meshresultstring+=wifimac+","+dst[1]+"/"
				if len(wifimactomac(dst[1]))>10{
					hopcount[wifimactomac(dst[1])]=1
				}

				//	fmt.Println(wifimac," to ",dst[1])
			}else{
				//	fmt.Println(wifimac)
				meshresultstring+=finddest(nxt[1],dst[1],wifimac,1)+"/"
			}
		}else{
		}

		//fmt.Println(mapdat[i])
	}
	mutex.Lock()
	meshmapstring[wifimac]=meshresultstring
	mutex.Unlock()
	//fmt.Println(meshresultstring)
}
func finddest(st string,des string,wifimac string,hop int) string {

	hop++
	returnvalue:=wifimac+","+st
	mac := strings.Replace(st, ":", "", -1)
	macTag := mac[len(mac)-6:]
//	fmt.Println(st,"/////여기???",macTag)
	findnextnode, err := readLines("./file_storage/gentelella-master/production/meshxml/"+macTag) //맵html파일
	if err != nil {
		fmt.Println("여기?")
		fmt.Println(err)
		return wifimac
	}
	compareroottime,err:=strconv.Atoi(findnextnode[len(findnextnode)-1])
	if err!=nil{
		fmt.Println(err)
	}
	fmt.Println(meshroottime-compareroottime,"비교한 초")
	if meshroottime-compareroottime>10{
		return wifimac
	}
	for i, _ := range findnextnode {
		if strings.Contains(findnextnode[i],",") {
			splitdata:=strings.Split(findnextnode[i],",")
			nxt:=strings.Split(splitdata[0],"=")
			dst:=strings.Split(splitdata[1],"=")
			if dst[1]==des{
				if(dst[1]==nxt[1]){
					if len(wifimactomac(dst[1]))>10 {
						returnvalue += "," + dst[1]
						hopcount[wifimactomac(des)] = hop
						return returnvalue
					}
					//meshresultstring+=wifimac+","+dst[1]+"/"
				}else{
					return finddest(nxt[1],dst[1],returnvalue,hop)
				}
			}
		}else{
	//	fmt.Println(findnextnode[i],"이건 초일걸")
		}
		//fmt.Println(mapdat[i])
	}
	return returnvalue
}
func findnextwifimac(mac string, def string)string{
//	fmt.Println(mac,"/",def)
	if def==""{
		return "0"
	}
	//fmt.Println(meshresultstring)
	res:=strings.SplitN(meshresultstring,"/",-1)
	for i:=0;i<len(res)-1;i++{
		if strings.Contains(res[i],mac){
			result:=strings.SplitN(res[i],",",-1)
			for j:=0;j<len(result);j++{
			//	fmt.Println(result[j],",",j,",",mac)
				if result[j]==mac{
					//fmt.Println(result[j-1],"<-",result[j])
					return strings.Split(strings.Split(def,result[j-1]+"/")[1],";")[0]
				}
			}
			//fmt.Println(mac,"/",res[i])
		}
	}
	return "0"
}
func meshinfo(wifimac string,str string,cnt int){

	mac := strings.Replace(wifimac, ":", "", -1)
	macTag := mac[len(mac)-6:]
	infostring:=strings.Split(str,",")
	//fmt.Println(macTag,str)
	//howmany := strings.Count(str, ",")
	//halfhowmany:=howmany/2
	var resultstring string
	resultstring=""
	var dststring [32]string
	var nxtstring [32]string
//	var dst [32]string
//	var nxt [32]string
	for i:=0;i<cnt;i++{
		dststring[i]=fmt.Sprint("meshDstMac",i)
		nxtstring[i]=fmt.Sprint("meshNxtMac",i)
		for j:=0;j<cnt*2;j++{
			if strings.Contains(infostring[j],nxtstring[i]){
				//fmt.Print(nxtstring[i],"=",strings.Split(infostring[j],"=")[1])
				resultstring+=nxtstring[i]+"="+strings.Split(infostring[j],"=")[1]+","
			}
		}
		for j:=0;j<cnt*2;j++{
			if strings.Contains(infostring[j],dststring[i]){
				resultstring+=dststring[i]+"="+strings.Split(infostring[j],"=")[1]+"\n"
			}
		}

	}
	resultstring+=strconv.Itoa(spend_time)

	//fmt.Println(dststring,"comma",nxtstring)
	//fmt.Println(resultstring)
	//lineArray2:=fmt.Println(lineArray[0])
	ioutil.WriteFile("./file_storage/gentelella-master/production/meshxml/"+macTag,[]byte(resultstring),os.FileMode(644))
}
var startline="<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<APTEST>\n"
var endline="</APTEST>"

func parsfunction(/*meshid,mac,conmac,ip,power*/psmeshid string,ip string,mac string,wifimac string,sig string){
	var totalstring string
	var checknum int
	if(len(mac)!=17){
		return
	}
	//fmt.Println(psmeshid)
	// xml 파일 오픈
	fp, err := os.Open("./file_storage/gentelella-master/production/meshxml/"+psmeshid+".xml")
	if err != nil {
		ioutil.WriteFile("./file_storage/gentelella-master/production/meshxml/"+psmeshid+".xml",[]byte(startline+"\t<Equip>\n\t\t<MAC>"+mac+"</MAC>\n\t\t<WIFIMAC>"+wifimac+"</WIFIMAC>\n\t\t<HEADING>"+wifimac+"</HEADING>\n\t\t<X>"+"0"+"</X>\n\t\t<Y>"+"0"+"</Y>\n\t\t<IP>"+ip+"</IP>\n\t\t<LINKM>"+sig+"</LINKM>\n\t</Equip>\n"+endline),os.FileMode(644))
		fmt.Println(err)
		return
	}
	defer fp.Close()
	// xml 파일 읽기
	data, err := ioutil.ReadAll(fp)
	//fmt.Println(sig,"신호세기")

	// xml 디코딩
	var equip1 APTEST
	xmlerr := xml.Unmarshal(data, &equip1)
	if xmlerr != nil {
		fp,err=os.Open("./file_storage/gentelella-master/production/meshxml/"+psmeshid+".xml.backup")
		if err != nil {
			panic(err)
		}
		data, err = ioutil.ReadAll(fp)
		xmlerr = xml.Unmarshal(data, &equip1)
		if xmlerr != nil {
			//fp.Close()
			//fmt.Println("지우고 다시시작")
			//err=os.Remove("sample1.xml")
			//fmt.Println(err)
			return
		}
		defer fp.Close()
		//ioutil.WriteFile()
	}
	totalstring=startline
	for i:=0;i<len(equip1.Equip);i++{
		if equip1.Equip[i].MAC==mac{ //해당하는 맥이면
			//fmt.Println(equip1.Equip[i].HEADING)c
			checknum++
			if sig=="err"{
				totalstring+="\t<Equip>\n\t\t<MAC>"+mac+"</MAC>\n\t\t<WIFIMAC>"+equip1.Equip[i].HEADING+"</WIFIMAC>\n\t\t<HEADING>"+equip1.Equip[i].HEADING+"</HEADING>\n\t\t<X>"+equip1.Equip[i].X+"</X>\n\t\t<Y>"+equip1.Equip[i].Y+"</Y>\n\t\t<IP>"+equip1.Equip[i].IP+"</IP>\n\t\t<LINKM>1010</LINKM>\n\t</Equip>\n"
			}else{
				totalstring+="\t<Equip>\n\t\t<MAC>"+mac+"</MAC>\n\t\t<WIFIMAC>"+wifimac+"</WIFIMAC>\n\t\t<HEADING>"+wifimac+"</HEADING>\n\t\t<X>"+equip1.Equip[i].X+"</X>\n\t\t<Y>"+equip1.Equip[i].Y+"</Y>\n\t\t<IP>"+ip+"</IP>\n\t\t<LINKM>"+sig+"</LINKM>\n\t</Equip>\n"

			}

		}else{
			totalstring+="\t<Equip>\n\t\t<MAC>"+equip1.Equip[i].MAC+"</MAC>\n\t\t<WIFIMAC>"+equip1.Equip[i].HEADING+"</WIFIMAC>\n\t\t<HEADING>"+equip1.Equip[i].HEADING+"</HEADING>\n\t\t<X>"+equip1.Equip[i].X+"</X>\n\t\t<Y>"+equip1.Equip[i].Y+"</Y>\n\t\t<IP>"+equip1.Equip[i].IP+"</IP>\n\t\t<LINKM>"+equip1.Equip[i].LINKM+"</LINKM>\n\t</Equip>\n"
		}
	}
	if checknum==0{
		totalstring+="\t<Equip>\n\t\t<MAC>"+mac+"</MAC>\n\t\t<WIFIMAC>"+wifimac+"</WIFIMAC>\n\t\t<HEADING>"+wifimac+"</HEADING>\n\t\t<X>0</X>\n\t\t<Y>0</Y>\n\t\t<IP>"+ip+"</IP>\n\t\t<LINKM>"+sig+"</LINKM>\n\t</Equip>\n"
		fmt.Println("태그없다" +
			"")
	}
	totalstring=totalstring+endline
	ioutil.WriteFile("./file_storage/gentelella-master/production/meshxml/"+psmeshid+".xml",[]byte(totalstring),os.FileMode(644))		//현재 데이터
	ioutil.WriteFile("./file_storage/gentelella-master/production/meshxml/"+psmeshid+".xml.backup",[]byte(data),os.FileMode(644))	// 직전 데이터

	//fmt.Println(totalstring)
}
func savemesh(w http.ResponseWriter, r *http.Request){
	var totalstring string
	r.ParseForm()
	a:=r.Form.Get("meshid")
	datastring:=r.Form.Get("data")
	datastringA:=strings.Split(datastring,"/")
	fmt.Println(a,datastringA)
	fp, err := os.Open("./file_storage/gentelella-master/production/meshxml/"+a+".xml")
	if err != nil {
		fmt.Println(err)
		//ioutil.WriteFile("./file_storage/gentelella-master/production/meshxml/"+psmeshid+".xml",[]byte(startline+"\t<Equip>\n\t\t<MAC>"+mac+"</MAC>\n\t\t<HEADING>"+head+"</HEADING>\n\t\t<X>"+"0"+"</X>\n\t\t<Y>"+"0"+"</Y>\n\t\t<IP>"+ip+"</IP>\n\t\t<LINKM>"+"1"+"</LINKM>\n\t</Equip>\n"+endline),os.FileMode(644))
		return
	}
	defer fp.Close()
	// xml 파일 읽기
	data, err := ioutil.ReadAll(fp)
	//fmt.Println(string(data))
	// xml 디코딩
	var equip1 APTEST
	xmlerr := xml.Unmarshal(data, &equip1)
	if xmlerr != nil {
		//fp,err=os.Open("./file_storage/gentelella-master/production/meshxml/"+psmeshid+".xml.backup")
		if err != nil {
			panic(err)
		}
		/*data, err = ioutil.ReadAll(fp)
		xmlerr = xml.Unmarshal(data, &equip1)
		if xmlerr != nil {
			//fp.Close()
			//fmt.Println("지우고 다시시작")
			//err=os.Remove("sample1.xml")
			//fmt.Println(err)
			return
		}
		defer fp.Close()*/
		//ioutil.WriteFile()
	}
	totalstring=startline
	for k:=0; k<len(datastringA);k++{
		data:=strings.Split(datastringA[k],",")
		fmt.Println(data[0])
		for i:=0;i<len(equip1.Equip);i++{
			if equip1.Equip[i].MAC==data[0]{ //해당하는 맥이면
				//fmt.Println(equip1.Equip[i].HEADING)c
				totalstring+="\t<Equip>\n\t\t<MAC>"+data[0]+"</MAC>\n\t\t<WIFIMAC>"+equip1.Equip[i].WIFIMAC+"</WIFIMAC>\n\t\t<HEADING>"+equip1.Equip[i].HEADING+"</HEADING>\n\t\t<X>"+data[1]+"</X>\n\t\t<Y>"+data[2]+"</Y>\n\t\t<IP>"+equip1.Equip[i].IP+"</IP>\n\t\t<LINKM>"+equip1.Equip[i].LINKM+"</LINKM>\n\t</Equip>\n"
			}
		}
	}
	totalstring=totalstring+endline
	ioutil.WriteFile("./file_storage/gentelella-master/production/meshxml/"+a+".xml",[]byte(totalstring),os.FileMode(644))		//현재 데이터
	//ioutil.WriteFile("./file_storage/gentelella-master/production/meshxml/"+psmeshid+".xml.backup",[]byte(data),os.FileMode(644))	// 직전 데이터

	if err!=nil{
		conf_table :=
			"<!DOCTYPE html> <html>	<html lang=\"en\"> <head><meta charset=\"UTF-8\"> </head> <body> <p><h1>JMP SYSTEMS Co., Ltd. 2017</h1></p>"
		conf_table += "</body> </html><script>window.onload=function(){alert('오류발생 관리자에게 문의해주세요.');window.close();}</script>"
		w.Write([]byte(conf_table))
		return
	}
	conf_table :=
		"<!DOCTYPE html> <html>	<html lang=\"en\"> <head><meta charset=\"UTF-8\"> </head> <body> <p><h1>JMP SYSTEMS Co., Ltd. 2017</h1></p>"
	conf_table += "</body> </html><script>window.onload=function(){alert('저장이 완료되었습니다.');window.close();}</script>"
	w.Write([]byte(conf_table))

}
var meshID string
var meshurl string
func showmeshid(w http.ResponseWriter, r *http.Request){
	r.ParseForm()//./file_storage/gentelella-master/production/
	meshID=r.Form.Get("meshid")
	meshurl=r.Form.Get("meshurl")
	fmt.Println(meshurl)
	conf_table :=
		"<!DOCTYPE html> <html>	<html lang=\"en\"> <head><meta charset=\"UTF-8\"> </head> <body> <p><h1>JMP SYSTEMS Co., Ltd. 2017</h1></p>"
	conf_table += "</body> </html><script>window.onload=function(){window.close();}</script>"
	w.Write([]byte(conf_table))
	//mesh 템플릿 만들어서 넣기
}
//ip=192.168.1.254&netmask=255.255.255.0&meshid=JMP-MESH&hold=-70&ipmode=false&rootmode=false
func savemeshconfig(w http.ResponseWriter, r *http.Request){
	netconfig:=""
	wireconfig:=""
	rootmode:="0"
	ann:="0"
	dhcpmode1:="static"
	dhcpmode2:="none"
	r.ParseForm()//./file_storage/gentelella-master/production/
	fmt.Println(r)
	macTag:=r.Form.Get("macTag")
	ip:=r.Form.Get("ip")
	netmask:=r.Form.Get("netmask")
	meshid:=r.Form.Get("meshid")
	tresh:=r.Form.Get("hold")
	dhcp:=r.Form.Get("ipmode")
	meshroot:=r.Form.Get("rootmode")
	gateway:=r.Form.Get("gateway")
	dns:=r.Form.Get("dns")
	fmt.Println("IP : ",ip," NETMASK : ", netmask," MESHID : ",meshid," TRESHOLD : ",tresh," DHCP : ",dhcp," MESHROOT : ",meshroot)
	if meshroot =="false"{
		rootmode="0"
		ann="0"
	}else{
		rootmode="4"
		ann="1"
	}
	if dhcp=="false"{
		dhcpmode1="static"
		dhcpmode2="none"
	}else{
		dhcpmode1="dhcp"
		dhcpmode2="none"
	}
	wireless, _ := readLines("./file_storage/cf_" + macTag + "/wireless")
	net, _ := readLines("./file_storage/cf_" + macTag + "/network")
	for i, _ := range wireless {
		if strings.Contains(wireless[i],"option mesh_rssi_threshold") {
			wireconfig+="	option mesh_rssi_threshold '"+tresh+"'\n"
		}else if strings.Contains(wireless[i],"option mesh_hwmp_rootmode"){
			wireconfig+="	option mesh_hwmp_rootmode '"+rootmode+"'\n"
		}else if strings.Contains(wireless[i],"option mesh_gate_announcements ") {
			wireconfig+="	option mesh_gate_announcements '"+ann+"'\n"
		}else if strings.Contains(wireless[i],"option mesh_id"){
			wireconfig+="	option mesh_id '"+meshid+"'\n"
		}else{
			wireconfig+=wireless[i]+"\n"
		}
		//fmt.Println(mapdat[i])
	}
	for i:=0;i<len(net);i++{
		if strings.Contains(net[i],"config interface 'loopback'"){
			netconfig+=net[i]+"\n"
			i++
			netconfig+=net[i]+"\n"
			i++
			netconfig+=net[i]+"\n"
			i++
			netconfig+=net[i]+"\n"
			i++
			netconfig+=net[i]+"\n"
		}else if strings.Contains(net[i],"option ipaddr "){
			netconfig+="	option ipaddr '"+ip+"'\n"
		}else if strings.Contains(net[i],"option netmask"){
			netconfig+="	option netmask '"+netmask+"'\n"
		}else if strings.Contains(net[i],"option proto "){
			netconfig+="	option proto '"+dhcpmode1+"'\n"
		}else if strings.Contains(net[i],"option autoip "){
			netconfig+="	option autoip '"+dhcpmode2+"'\n"
		}else if strings.Contains(net[i],"option gateway"){
			netconfig+="	option gateway '"+gateway+"'\n"
		}else if strings.Contains(net[i],"option dns"){
			netconfig+="	option dns '"+dns+"'\n"
		}else{
			netconfig+=net[i]+"\n"
		}
	}
	fmt.Println(netconfig)
	ioutil.WriteFile("./file_storage/cf_" + macTag + "/wireless",[]byte(wireconfig),os.FileMode(644))
	ioutil.WriteFile("./file_storage/cf_" + macTag + "/network",[]byte(netconfig),os.FileMode(644))
	go func() {
		if my7zip(macTag) != nil {
			// 혹시 외부함수로 서버 기능들 블럭 될까봐...
			print("압축 오류 발생 !!")
		}
	}()
}
func showmesh(w http.ResponseWriter, r *http.Request){
	r.ParseForm()
	//fmt.Println(meshID)
	/*
	conf_table, _ := ioutil.ReadFile("confedit.html")
	conf_template := template.New("ctempl")
	conf_template, _ = conf_template.Parse(string(conf_table))
	w.Header().Set("Content-Type", "text/html")
	*/
	mesh_table, _ := ioutil.ReadFile("./file_storage/gentelella-master/production/mesh.html")
	mesh_template := template.New("creattempl")
	mesh_template, _ = mesh_template.Parse(string(mesh_table))
	w.Header().Set("Content-Type", "text/html")
	meshid_parser := mesh{
		XmlFilename: meshID,
		Url			: meshurl,
	}
	executeErr := mesh_template.Execute(w, meshid_parser)
	fmt.Println(meshid_parser)
	if executeErr != nil {
		fmt.Println("Template Execution Error: ", executeErr)
	}
	//mesh 템플릿 만들어서 넣기
}

func editConfig(w http.ResponseWriter, r *http.Request) {
	openWRT :=false
	r.ParseForm()
	var dat string
	if dateRange, ok := r.Form["macAddr"]; !ok {
		fmt.Println("대상 AP MAC 주소가 없습니다.. 사용법 확인하세요", dateRange)
		fmt.Printf("%T  --------\n", dat)
		w.Write([]byte("쿼리에 AP MAC 주소가 없습니다.. 사용법 확인하세요"))
		return
	}
	db, err := sql.Open("mysql", "root:1111@tcp("+dbLocation+":"+db_port+")/map")
	if err != nil {
		panic(err)
	}
	// macTag 확정
	var macTag string
	var noZipFileFlag string
	var modelName string
	var conf_table string
	var config_string string
	var readDeault string
	var Default string

	saveFlag := r.Form.Get("saveFlag")
	saveFg := "" + saveFlag

	mac := r.Form.Get("macAddr")
	println("------------------->>>>", mac)
	macAddr := "" + mac
	mac = strings.Replace(mac, ":", "", -1)
	macTag = mac[len(mac)-6:]
	curYearMonth := time.Now().String()
	curYearBuf := curYearMonth[:4]                // 연도 짤라내기. 글자수는 0부터 시작하고 끝 숫자는 빠진다. 0,1,2,3 위치 잘라내기
	curYearMonth = curYearBuf + curYearMonth[5:7] // 월 짤라내기
	site := "site" + curYearMonth
	ipQuerySt1 := "SELECT modelName from " + site + " WHERE macAddr = " + "\"" + macAddr + "\"" /// 물어볼땐 기존 n 구조체 MAC을 사용 북마크북마크
	//fmt.Println("///////////111",ipQuerySt1)
	err = db.QueryRow(ipQuerySt1).Scan(&modelName) //

	fmt.Println("///////////////", modelName, "///////////////")
	cf_matrix0 := [][]string{} //변수길이를 못쓰니까 slice로 대체
	cf_matrix1 := [][]string{} //변수길이를 못쓰니까 slice로 대체
	cf_matrix2 := [][]string{} //변수길이를 못쓰니까 slice로 대체 (7500 때문에 총 3개)
	cf_map0 := map[string]string{}
	cf_map1 := map[string]string{}
	cf_map2 := map[string]string{}

	if saveFg == "save" {
		if modelName == "MAP7500" {
			cc1 := map[string]string{ // 첫번째 파일 수정용 map
				"IP_ADDRESS":      "",
				"SUBNET_MASK":     "",
				"DEFAULT_GATEWAY": "",
			}
			cc2 := map[string]string{ // 2번째 파일 수정용 map
				"ESSID[0]":           "",
				"WPAP_PASSPHRASE[0]": "",
				"SECURITY_MODE[0]":   "",
			}
			cc3 := map[string]string{ // 3번째 파일 수정용 map
				"ESSID[1]":           "",
				"WPAP_PASSPHRASE[1]": "",
				"SECURITY_MODE[1]":   "",
			}
			for k, v := range r.Form {
				tempV := strings.Join(v, "") // 문자 map value 값을 string으로 변환하기
				switch k {
				case "Lan_ipaddr":
					cc1["IP_ADDRESS"] = tempV
				case "Lan_netmask":
					cc1["SUBNET_MASK"] = tempV
				case "Lan_gateway":
					cc1["DEFAULT_GATEWAY"] = tempV
				case "SSID_2G":
					cc2["ESSID[0]"] = tempV
				case "WPAPSK_2G":
					cc2["WPAP_PASSPHRASE[0]"] = tempV
				case "SSID_5G":
					cc3["ESSID[1]"] = tempV
				case "WPAPSK_5G":
					cc3["WPAP_PASSPHRASE[1]"] = tempV
				case "Secure_Mode_2G":
					cc2["SECURITY_MODE[0]"] = tempV
				case "Secure_Mode_5G":
					cc3["SECURITY_MODE[1]"] = tempV
				default:
				}
			}
			//fmt.Println("+++++++++++++++++++", cc1,cc2)

			filePath0 := "./file_storage/cf_" + macTag + "/config" // 변경할 대상 파일 위치
			summaryData0 := cc1                                    // 변경할 내용
			firmWareStr0 := "7500"                                 // 파싱할 파일 스타일
			cf_matrix0 = updateConfigFile(filePath0, summaryData0, firmWareStr0)

			filePath1 := "./file_storage/cf_" + macTag + "/config.rf0" // 변경할 대상 파일 위치
			summaryData1 := cc2                                        // 변경할 내용
			firmWareStr1 := "7500"                                     // 파싱할 파일 스타일
			cf_matrix1 = updateConfigFile(filePath1, summaryData1, firmWareStr1)

			filePath2 := "./file_storage/cf_" + macTag + "/config.rf1" // 변경할 대상 파일 위치
			summaryData2 := cc3                                        // 변경할 내용
			firmWareStr2 := "7500"                                     // 파싱할 파일 스타일
			cf_matrix2 = updateConfigFile(filePath2, summaryData2, firmWareStr2)
			//fmt.Println(cf_matrix0, cf_matrix1)

			// 해당 mac 값의 파일
			//ArchiveFile("./file_storage/cf_" + macTag + "/", "./file_storage/cf_" + macTag + ".zip", nil )
			go func() {
				if my7zip(macTag) != nil {
					// 혹시 외부함수로 서버 기능들 블럭 될까봐...
					print("압축 오류 발생 !!")
				}
			}()

			textMsg0 := ""
			textMsg1 := ""
			textMsg2 := ""

			for i := 0; i < len(cf_matrix0); i++ {
				cf_map0[cf_matrix0[i][0]] = cf_matrix0[i][1]
				textMsg0 += cf_matrix0[i][0] + " : " + cf_matrix0[i][1] + "\n"
			}
			for i := 0; i < len(cf_matrix1); i++ {
				cf_map1[cf_matrix1[i][0]] = cf_matrix1[i][1]
				textMsg1 += cf_matrix1[i][0] + " : " + cf_matrix1[i][1] + "\n"
			}

			for i := 0; i < len(cf_matrix2); i++ {
				cf_map2[cf_matrix2[i][0]] = cf_matrix2[i][1]
				textMsg2 += cf_matrix2[i][0] + " : " + cf_matrix2[i][1] + "\n"
			}

			w.Header().Set("Content-Type", "text/html")
			conf_table =
				"<!DOCTYPE html> <html>	<html lang=\"en\"> <head> <style> table {width:100%;} table, th, td {" + " border: 1px solid black; border-collapse: collapse;	} th, td { padding: 5px; text-align: center;" + "} table.names tr:nth-child(even) { background-color: #eee; }	table.names tr:nth-child(odd) {" + "background-color:#fff; } table.names th { background-color: black; color: yellow } </style>" + "<meta charset=\"UTF-8\"> </head> <body> <p>JMP SYSTEMS Co., Ltd. 2017</p><p></p><p></p><p></p><table class=\"names\"> <tr> <th>Zip 저장완료 </th>" + "<th>MAC: " + macAddr + "</th> </tr>"
			conf_table += "</table><div><p></p>ctl+w 또는 닫기 클릭하세요 <button onclick=\"self.close()\"> 닫기 </button><p></p></div>"
			conf_table += "<p>메인보드 유선설정</p>"
			conf_table += "<textarea readonly=\"readonly\" onclick=\"this.select()\" style= \"background-color:#f9e0fF; min-width:100%; max-width:100%; min-height:256px; max-height:256px\" >" + textMsg0 + "</textarea></br>"
			conf_table += "<p>무선1</p>"
			conf_table += "<textarea readonly=\"readonly\" onclick=\"this.select()\" style= \"background-color:#e9f5fF; min-width:100%; max-width:100%; min-height:256px; max-height:256px\" >" + textMsg1 + "</textarea></br>"
			conf_table += "<p>무선2</p>"
			conf_table += "<textarea readonly=\"readonly\" onclick=\"this.select()\" style= \"background-color:#e9e5ff; min-width:100%; max-width:100%; min-height:256px; max-height:256px\" >" + textMsg2 + "</textarea></br>"
			conf_table += "</body> </html>"

			w.Write([]byte(conf_table))
			w.Write([]byte(macAddr + " 장비의 설정파일이 zip으로 저장되었습니다\r\n"))

		} else if modelName == "MAP7700" || modelName == "MAP9220" || modelName == "MAP9200" || modelName == "MAP5010" || modelName == "MAP2010" { //장비추가////
			cc1 := map[string]string{
				"lan_ipaddr":  "",
				"lan_netmask": "",
				"wan_gateway": "",
				"SSID1":       "",
				"WPAPSK1":     "",
				"AuthMode":    "",
				"EncrypType":  "",
				"curl_server_url": "",
			}
			cc2 := map[string]string{
				"SSID1":      "",
				"WPAPSK1":    "",
				"AuthMode":   "",
				"EncrypType": "",
			}
			for k, v := range r.Form {
				tempV := strings.Join(v, "") // 문자 map value 값을 string으로 변환하기
				switch k {
				case "Lan_ipaddr":
					cc1["lan_ipaddr"] = tempV
				case "Lan_netmask":
					cc1["lan_netmask"] = tempV
				case "Lan_gateway":
					cc1["wan_gateway"] = tempV
				case "SSID_2G":
					cc1["SSID1"] = tempV
				case "WPAPSK_2G":
					cc1["WPAPSK1"] = tempV
				case "SSID_5G":
					cc2["SSID1"] = tempV
				case "WPAPSK_5G":
					cc2["WPAPSK1"] = tempV
				case "Secure_Mode_2G":
					cc1["AuthMode"] = tempV
				case "Secure_Mode_enc_2G":
					cc1["EncrypType"] = tempV
				case "Secure_Mode_5G":
					cc2["AuthMode"] = tempV
				case "Secure_Mode_enc_5G":
					cc2["EncrypType"] = tempV
				case "curl_server_url":
					cc1["curl_server_url"] = tempV
				default:
				}
			}
			//fmt.Println("+++++++++++++++++++", cc1,cc2)
			filePath0 := "./file_storage/cf_" + macTag + "/RT2880_Settings.dat" // 변경할 대상 파일 위치
			summaryData0 := cc1                                                 // 변경할 내용
			firmWareStr0 := "7700_RT2880_Settings"                              // 파싱할 파일 스타일
			cf_matrix0 = updateConfigFile(filePath0, summaryData0, firmWareStr0)

			filePath1 := "./file_storage/cf_" + macTag + "/iNIC_ap.dat" // 변경할 대상 파일 위치
			summaryData1 := cc2                                         // 변경할 내용
			firmWareStr1 := "7700_iNIC_ap"                              // 파싱할 파일 스타일
			cf_matrix1 = updateConfigFile(filePath1, summaryData1, firmWareStr1)

			//fmt.Println(cf_matrix0, cf_matrix1)

			// 해당 mac 값의 파일
			//ArchiveFile("./file_storage/cf_" + macTag + "/", "./file_storage/cf_" + macTag + ".zip", nil )
			go func() {
				if my7zip(macTag) != nil {
					// 혹시 외부함수로 서버 기능들 블럭 될까봐...
					print("압축 오류 발생 !!")
				}
			}()
			textMsg0 := ""
			textMsg1 := ""
			for i := 0; i < len(cf_matrix0); i++ {
				cf_map0[cf_matrix0[i][0]] = cf_matrix0[i][1]
				textMsg0 += cf_matrix0[i][0] + " : " + cf_matrix0[i][1] + "\n"
			}
			for i := 0; i < len(cf_matrix1); i++ {
				cf_map1[cf_matrix1[i][0]] = cf_matrix1[i][1]
				textMsg1 += cf_matrix1[i][0] + " : " + cf_matrix1[i][1] + "\n"
			}
			w.Header().Set("Content-Type", "text/html")
			conf_table =
				"<!DOCTYPE html> <html>	<html lang=\"en\"> <head> <style> table {width:100%;} table, th, td {" + " border: 1px solid black; border-collapse: collapse;	} th, td { padding: 5px; text-align: center;" + "} table.names tr:nth-child(even) { background-color: #eee; }	table.names tr:nth-child(odd) {" + "background-color:#fff; } table.names th { background-color: black; color: yellow } </style>" + "<meta charset=\"UTF-8\"> </head> <body> <p>JMP SYSTEMS Co., Ltd. 2017</p><p></p><p></p><p></p><table class=\"names\"> <tr> <th>Zip 저장완료 </th>" + "<th>MAC: " + macAddr + "</th> </tr>"
			conf_table += "</table><div><p></p>ctl+w 또는 닫기 클릭하세요 <button onclick=\"self.close()\"> 닫기 </button><p></p></div>"
			conf_table += "<p>RT2880Setting.dat</p>"
			conf_table += "<textarea readonly=\"readonly\" onclick=\"this.select()\" style= \"background-color:#D9E5FF; min-width:100%; max-width:100%; min-height:256px; max-height:256px\" >" + textMsg0 + "</textarea></br>"
			conf_table += "<p>iNIC_ap.dat</p>"
			conf_table += "<textarea readonly=\"readonly\" onclick=\"this.select()\" style= \"background-color:#e9f5fF; min-width:100%; max-width:100%; min-height:256px; max-height:256px\" >" + textMsg1 + "</textarea></br>"
			conf_table += "</body> </html>"

			w.Write([]byte(conf_table))
			w.Write([]byte(macAddr + " 장비의 설정파일이 zip으로 저장되었습니다\r\n"))
		}
	} else {
		// 일단 모델무관 파일 존재 확인 및 압축 풀어 놓기
		_, err := ioutil.ReadFile("./file_storage/cf_" + macTag + ".zip") // 파일 유무 확인
		if err != nil {
			fmt.Println(err)
			readDeault = "해당장비의 설정파일이 없습니다. 장비의 설정을 불러오거나 템플렛을 사용합니다."
			fmt.Print("해당 장비 설정 파일이 없습니다. default 설정파일을 읽어 옵니다\n")
			if modelName == "MAP7500" {
				_, err = ioutil.ReadFile("./file_storage/cf_default7500.zip")
				if err != nil {
					print("cf_default7500.zip 파일도 없습니다. 시스템 점검이 필요합니다\n")
					os.Exit(7878)
				}
			} else if modelName == "MAP7700" || modelName == "MAP9220" || modelName == "MAP9200" || modelName == "MAP5010" || modelName == "MAP2010" { //장비추가///
				_, err = ioutil.ReadFile("./file_storage/cf_default.zip")
				if err != nil {
					print("cf_default.zip 파일도 없습니다. 시스템 점검이 필요합니다\n")
					os.Exit(8888)
				}
			} else if modelName =="5000_MESH" {
				_, err = ioutil.ReadFile("./file_storage/cf_defaultmesh.zip")
				fmt.Println("여기여기여기 메쉬 파일 여기있다")
				if err != nil {
					print("cf_default.zip 파일도 없습니다. 시스템 점검이 필요합니다\n")
					os.Exit(8888)
				}

			}else{
				println("\n모델명이 잘못되었습니다. MAC 주소 확인바랍니다..\n")
				w.Write([]byte("모델명이 잘못되었습니다. 펌웨어 업데이트가 필요합니다."))
				return // 모델명 못찾으면 리턴해야 한다
			}
			noZipFileFlag = "DEFAULT"
		}
		// 어느 경우든지 macTag에 해당하는 파일 디렉토리 & 설정 파일들을 만든다.
		if noZipFileFlag == "DEFAULT" { // 디폴트 파일을 해당 MAC으로 풀어 놓는다.. 자가 증식 ~~
			if modelName == "MAP7500" {
				err = Unzip("./file_storage/cf_default7500.zip", "./file_storage/cf_"+macTag)
				if err != nil {
					fmt.Println(err)
				}
			} else if modelName == "MAP7700" || modelName == "MAP9220" || modelName == "MAP9200" || modelName == "MAP5010" || modelName == "MAP2010"{//장비추가
				err = Unzip("./file_storage/cf_default.zip", "./file_storage/cf_"+macTag)
				if err != nil {
					fmt.Println(err)
				}
			}else if modelName=="5000_MESH"{
				Default = "저장된 설정 파일이 없습니다.\n반드시 설정 변경을 해주세요."
				err = Unzip("./file_storage/cf_defaultmesh.zip", "./file_storage/cf_"+macTag)
				if err != nil {
					fmt.Println(err)
				}
			}else {
				println("\n모델명 분기 에러 입니다. MAC 주소 확인바랍니다..\n")
				return // 모델명 못찾으면 리턴해야 한다
			}
		} else { // 직전 zip으로 만든 파일을 풀어 놓는다.
			err = Unzip("./file_storage/cf_"+macTag+".zip", "./file_storage/cf_"+macTag)
			if err != nil {
				fmt.Println(err)
			}
		}
		// 모델에 따라 설정 슬라이스 가져오기
		if modelName == "MAP7500" {
			filePath0 := "./file_storage/cf_" + macTag + "/config" // 변경할 대상 파일 위치
			summaryData0 := map[string]string{"": ""}
			firmWareStr0 := "7500" // 파싱할 파일 스타일
			cf_matrix0 = updateConfigFile(filePath0, summaryData0, firmWareStr0)

			filePath1 := "./file_storage/cf_" + macTag + "/config.rf0" // 변경할 대상 파일 위치
			summaryData1 := map[string]string{"": ""}
			firmWareStr1 := "7500" // 파싱할 파일 스타일
			cf_matrix1 = updateConfigFile(filePath1, summaryData1, firmWareStr1)

			filePath2 := "./file_storage/cf_" + macTag + "/config.rf1" // 변경할 대상 파일 위치
			summaryData2 := map[string]string{"": ""}
			firmWareStr2 := "7500" // 파싱할 파일 스타일
			cf_matrix2 = updateConfigFile(filePath2, summaryData2, firmWareStr2)
			for i := 0; i < len(cf_matrix0); i++ {
				cf_map0[cf_matrix0[i][0]] = cf_matrix0[i][1]
				config_string += cf_matrix0[i][0] + " : " + cf_matrix0[i][1] + "\n"
			}
			config_string += "\n#####################################\n"
			for i := 0; i < len(cf_matrix1); i++ {
				cf_map1[cf_matrix1[i][0]] = cf_matrix1[i][1]
				config_string += cf_matrix1[i][0] + " : " + cf_matrix1[i][1] + "\n"
			}
			config_string += "\n#####################################\n"
			for i := 0; i < len(cf_matrix2); i++ {
				cf_map2[cf_matrix2[i][0]] = cf_matrix2[i][1]
				config_string += cf_matrix2[i][0] + " : " + cf_matrix2[i][1] + "\n"
			}

		} else if modelName == "MAP7700" || modelName == "MAP9220" || modelName == "MAP9200" || modelName == "MAP5010" || modelName == "MAP2010" { //장비추가
			filePath0 := "./file_storage/cf_" + macTag + "/RT2880_Settings.dat" // 변경할 대상 파일 위치
			summaryData0 := map[string]string{"": ""}
			firmWareStr0 := "7700_RT2880_Settings" // 파싱할 파일 스타일
			cf_matrix0 = updateConfigFile(filePath0, summaryData0, firmWareStr0)

			filePath1 := "./file_storage/cf_" + macTag + "/iNIC_ap.dat" // 변경할 대상 파일 위치
			summaryData1 := map[string]string{"": ""}
			firmWareStr1 := "7700_iNIC_ap" // 파싱할 파일 스타일
			cf_matrix1 = updateConfigFile(filePath1, summaryData1, firmWareStr1)

			for i := 0; i < len(cf_matrix0); i++ {
				cf_map0[cf_matrix0[i][0]] = cf_matrix0[i][1]
				config_string += cf_matrix0[i][0] + " : " + cf_matrix0[i][1] + "\n"
			}
			config_string += "\n#####################################\n"
			for i := 0; i < len(cf_matrix1); i++ {
				cf_map1[cf_matrix1[i][0]] = cf_matrix1[i][1]
				config_string += cf_matrix1[i][0] + " : " + cf_matrix1[i][1] + "\n"
			}
		}else if modelName =="5000_MESH"{
			//w.Write([]byte("5000 메쉬입니다."))
			openWRT=true
		}
		if(openWRT){

			configmeshmap := make(map[string]string)    //설정 담을 map
			net, _ := readLines("./file_storage/cf_" + macTag + "/network")
			for i, _ := range net {
				if strings.Contains(net[i],"option") {
					res:=strings.SplitN(net[i],"'",-1)
					configmeshmap[res[0]]=res[1]
				}
				//fmt.Println(mapdat[i])
			}

			wire,_:=readLines("./file_storage/cf_" + macTag + "/wireless")
			for i, _ := range wire {
				if strings.Contains(wire[i],"option") {
					res:=strings.SplitN(wire[i],"'",-1)
					configmeshmap[res[0]]=res[1]
				}
				//fmt.Println(mapdat[i])
			}
			fmt.Println(configmeshmap["	option netmask "])
			//res:=fmt.Sprint(string(net),"\n===========\n",string(wire))
			//w.Write([]byte(res))
			conf_table, _ := ioutil.ReadFile("meshedit.html")
			conf_template := template.New("ctempl")
			conf_template, _ = conf_template.Parse(string(conf_table))
			w.Header().Set("Content-Type", "text/html")
			conf_mesh:=meshStruct{
				ReadDefaultFile:  Default,
				Dhcp:configmeshmap["	option proto "],
				Ipaddr:configmeshmap["	option ipaddr "],
				Netmask:configmeshmap["	option netmask "],
				Meshid:	configmeshmap["	option mesh_id "],
				Gateway:configmeshmap["	option gateway "],
				Dns:configmeshmap["	option dns "],
				Threshold:	configmeshmap["	option mesh_rssi_threshold "],
				Rootmode:	configmeshmap["	option mesh_hwmp_rootmode "],
				Modelname:modelName,
				Mactag:macTag,
				Wireless:configmeshmap["	option mesh_id "],
				Network:configmeshmap["	option netmask "],
			}
			executeErr := conf_template.Execute(w, conf_mesh)
			if executeErr != nil {
				fmt.Println("Template Execution Error: ", executeErr)
			}
			return
		}
		// 브라우저에 hmtl 뿌려주기
		conf_table, _ := ioutil.ReadFile("confedit.html")
		conf_template := template.New("ctempl")
		conf_template, _ = conf_template.Parse(string(conf_table))
		w.Header().Set("Content-Type", "text/html")
		// 제품별로 주요 정보를 수집하여 대표 변수에 입력 시켜야 한다
		switch modelName {
		case "MAP7500":
			conf_7500 := configStruct{
				ReadDefaultFile:  readDeault,
				ModelName:        modelName,
				MacAddr:          macAddr,
				Lan_ipaddr:       cf_map0["IP_ADDRESS"], // 메인보드 온리
				Lan_netmask:      cf_map0["SUBNET_MASK"],
				Lan_gateway:      cf_map0["DEFAULT_GATEWAY"],
				SSID_2G:          cf_map1["ESSID[0]"], // 무선 1
				WPAPSK_2G:        cf_map1["WPAP_PASSPHRASE[0]"],
				SSID_5G:          cf_map2["ESSID[1]"], // 무선 2
				WPAPSK_5G:        cf_map2["WPAP_PASSPHRASE[1]"],
				Security_mode_2G: cf_map1["SECURITY_MODE[0]"],
				Security_mode_5G: cf_map2["SECURITY_MODE[1]"],
				Config_string: config_string,
			}
			//fmt.Println(conf)
			executeErr := conf_template.Execute(w, conf_7500)
			if executeErr != nil {
				fmt.Println("Template Execution Error: ", executeErr)
			}

		case "MAP7700":
			conf_7700 := configStruct{
				ReadDefaultFile:      readDeault,
				ModelName:            modelName,
				MacAddr:              macAddr,
				Lan_ipaddr:           cf_map0["lan_ipaddr"], // 메인보드와 무선 1
				Lan_netmask:          cf_map0["lan_netmask"],
				Lan_gateway:          cf_map0["wan_gateway"],
				SSID_2G:              cf_map0["SSID1"],
				WPAPSK_2G:            cf_map0["WPAPSK1"],
				SSID_5G:              cf_map1["SSID1"], // 무선 2 -- 5기가
				WPAPSK_5G:            cf_map1["WPAPSK1"],
				Security_mode_2G:     cf_map0["AuthMode"],
				Security_mode_5G:     cf_map1["AuthMode"],
				Security_mode_enc_2G: cf_map0["EncrypType"],
				Security_mode_enc_5G: cf_map1["EncrypType"],
				CURL :cf_map0["curl_server_url"],
				Config_string:        config_string,
			}
			//fmt.Println(conf)
			executeErr := conf_template.Execute(w, conf_7700)
			if executeErr != nil {
				fmt.Println("Template Execution Error: ", executeErr)

			}
		case "MAP9220":
			conf_9220 := configStruct{
				ReadDefaultFile:      readDeault,
				ModelName:            "MAP9220",
				MacAddr:              macAddr,
				Lan_ipaddr:           cf_map0["lan_ipaddr"], // 메인보드와 무선 1
				Lan_netmask:          cf_map0["lan_netmask"],
				Lan_gateway:          cf_map0["wan_gateway"],
				SSID_2G:              cf_map0["SSID1"],
				WPAPSK_2G:            cf_map0["WPAPSK1"],
				SSID_5G:              cf_map1["SSID1"], // 무선 2 -- 5기가
				WPAPSK_5G:            cf_map1["WPAPSK1"],
				Security_mode_2G:     cf_map0["AuthMode"],
				Security_mode_5G:     cf_map1["AuthMode"],
				Security_mode_enc_2G: cf_map0["EncrypType"],
				Security_mode_enc_5G: cf_map1["EncrypType"],
				CURL :cf_map0["curl_server_url"],
				Config_string:        config_string,
			}
			fmt.Println(cf_map0["curl_server_url"])
			//fmt.Println(conf)
			executeErr := conf_template.Execute(w, conf_9220)
			if executeErr != nil {
				fmt.Println("Template Execution Error: ", executeErr)

			}
		case "MAP9200":
			conf_9200 := configStruct{
				ReadDefaultFile:      readDeault,
				ModelName:            modelName,
				MacAddr:              macAddr,
				Lan_ipaddr:           cf_map0["lan_ipaddr"], // 메인보드와 무선 1
				Lan_netmask:          cf_map0["lan_netmask"],
				Lan_gateway:          cf_map0["wan_gateway"],
				SSID_2G:              cf_map0["SSID1"],
				WPAPSK_2G:            cf_map0["WPAPSK1"],
				SSID_5G:              cf_map1["SSID1"], // 무선 2 -- 5기가
				WPAPSK_5G:            cf_map1["WPAPSK1"],
				Security_mode_2G:     cf_map0["AuthMode"],
				Security_mode_5G:     cf_map1["AuthMode"],
				Security_mode_enc_2G: cf_map0["EncrypType"],
				Security_mode_enc_5G: cf_map1["EncrypType"],
				CURL :cf_map0["curl_server_url"],
				Config_string:        config_string,
			}
			//fmt.Println(conf)
			executeErr := conf_template.Execute(w, conf_9200)
			if executeErr != nil {
				fmt.Println("Template Execution Error: ", executeErr)

			}
		case "MAP5010":
			conf_5000 := configStruct{
				ReadDefaultFile: readDeault,
				ModelName:       modelName,
				MacAddr:         macAddr,
				Lan_ipaddr:      cf_map0["lan_ipaddr"], // 메인보드와 무선 1
				Lan_netmask:     cf_map0["lan_netmask"],
				Lan_gateway:     cf_map0["wan_gateway"],
				//SSID_2G: cf_map0["SSID1"],
				//WPAPSK_2G: cf_map0["WPAPSK1"],
				SSID_5G:   cf_map1["SSID1"], // 무선 2 -- 5기가
				WPAPSK_5G: cf_map1["WPAPSK1"],
				//Security_mode_2G:cf_map0["AuthMode"],
				Security_mode_5G: cf_map1["AuthMode"],
				//Security_mode_enc_2G:cf_map0["EncrypType"],
				Security_mode_enc_5G: cf_map1["EncrypType"],
				CURL :cf_map0["curl_server_url"],
				Config_string:        config_string,
			}
			//fmt.Println(conf)
			executeErr := conf_template.Execute(w, conf_5000)
			if executeErr != nil {
				fmt.Println("Template Execution Error: ", executeErr)
			}
		case "MAP2010":
			conf_2000 := configStruct{
				ReadDefaultFile: readDeault,
				ModelName:       modelName,
				MacAddr:         macAddr,
				Lan_ipaddr:      cf_map0["lan_ipaddr"], // 메인보드와 무선 1
				Lan_netmask:     cf_map0["lan_netmask"],
				Lan_gateway:     cf_map0["wan_gateway"],
				//SSID_2G: cf_map0["SSID1"],
				//WPAPSK_2G: cf_map0["WPAPSK1"],
				SSID_5G:   cf_map1["SSID1"], // 무선 2 -- 5기가
				WPAPSK_5G: cf_map1["WPAPSK1"],
				//Security_mode_2G:cf_map0["AuthMode"],
				Security_mode_5G: cf_map1["AuthMode"],
				//Security_mode_enc_2G:cf_map0["EncrypType"],
				Security_mode_enc_5G: cf_map1["EncrypType"],
				CURL :cf_map0["curl_server_url"],
				Config_string:        config_string,
			}
			//fmt.Println(conf)
			executeErr := conf_template.Execute(w, conf_2000)
			if executeErr != nil {
				fmt.Println("Template Execution Error: ", executeErr)
			}
		}
		//config_string := ""
	}
}
func setMultiConfig(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var dat string
	//var err error
	db, err := sql.Open("mysql", "root:1111@tcp("+dbLocation+":"+db_port+")/map")
	if err != nil {
		panic(err)
	}
	//st = "site" + curYearMonth // ""사이트"" 테이블명 확정
	//db.Exec("TRUNCATE TABLE "+st)
	defer db.Close()

	chke := r.Form.Get("Num_update")
	checked := "" + chke
	//fmt.Println(chke)
	var howmany int
	howmany, _ = strconv.Atoi(checked)
	//fmt.Println(howmany, err)

	if dateRange, ok := r.Form["macAddr1"]; !ok {
		fmt.Println("대상 AP MAC 주소가 없습니다.. 사용법 확인하세요", dateRange)
		fmt.Printf("%T  --------\n", dat)
		w.Write([]byte("쿼리에 AP MAC 주소가 없습니다.. 사용법 확인하세요"))
		return
	}
	// macTag 확정
	var macTag string
	var noZipFileFlag string
	var modelName string
	var gateway string

	var config_string string
	var readDeault string

	var mac11 string
	var mac22 string
	var mac33 string
	var mac44 string
	var mac55 string
	var mac66 string
	var mac77 string
	var mac88 string
	var mac99 string
	var mac10 string

	var ip11 string
	var ip22 string
	var ip33 string
	var ip44 string
	var ip55 string
	var ip66 string
	var ip77 string
	var ip88 string
	var ip99 string
	var ip10 string

	var pass11 string
	var pass22 string
	var pass33 string
	var pass44 string
	var pass55 string
	var pass66 string
	var pass77 string
	var pass88 string
	var pass99 string
	var pass10 string

	var id11 string
	var id22 string
	var id33 string
	var id44 string
	var id55 string
	var id66 string
	var id77 string
	var id88 string
	var id99 string
	var id10 string

	var pass5g11 string
	var pass5g22 string
	var pass5g33 string
	var pass5g44 string
	var pass5g55 string
	var pass5g66 string
	var pass5g77 string
	var pass5g88 string
	var pass5g99 string
	var pass5g10 string

	var id5g11 string
	var id5g22 string
	var id5g33 string
	var id5g44 string
	var id5g55 string
	var id5g66 string
	var id5g77 string
	var id5g88 string
	var id5g99 string
	var id5g10 string

	var curl string

	macarray := []string{}

	cf_matrix0 := [][]string{} //변수길이를 못쓰니까 slice로 대체
	cf_matrix1 := [][]string{} //변수길이를 못쓰니까 slice로 대체
	cf_matrix2 := [][]string{} //변수길이를 못쓰니까 slice로 대체 (7500 때문에 총 3개)
	cf_map0 := map[string]string{}
	cf_map1 := map[string]string{}
	cf_map2 := map[string]string{}

	conf_template := template.New("ctempl")                     //템플릿 만들고
	conf_table1, _ := ioutil.ReadFile("newtemp.html")           //템플릿 적용할 파일 불러들이고
	conf_template, _ = conf_template.Parse(string(conf_table1)) // html 파일을 템플릿 obj에 담는다
	w.Header().Set("Content-Type", "text/html")

	var cf [20]configStruct

	var s1 string

	for i := 0; i < howmany; i++ {

		saveFlag := r.Form.Get("saveFlag")
		saveFg := "" + saveFlag
		s1 = fmt.Sprint("macAddr", i+1)
		mac := r.Form.Get(s1)

		println("------------------->>>>", mac)
		macAddr := "" + mac

		macarray = append(macarray, macAddr)
		mac = strings.Replace(mac, ":", "", -1)
		macTag = mac[len(mac)-6:]
		curYearMonth := time.Now().String()
		curYearBuf := curYearMonth[:4]                // 연도 짤라내기. 글자수는 0부터 시작하고 끝 숫자는 빠진다. 0,1,2,3 위치 잘라내기
		curYearMonth = curYearBuf + curYearMonth[5:7] // 월 짤라내기
		site := "site" + curYearMonth
		ipQuerySt1 := "SELECT modelName from " + site + " WHERE macAddr = " + "\"" + macAddr + "\"" /// 물어볼땐 기존 n 구조체 MAC을 사용 북마크북마크
		//fmt.Println("///////////111",ipQuerySt1)
		err = db.QueryRow(ipQuerySt1).Scan(&modelName) // s 구조체변수로 받아낸다
		//modelName으로 이제 판별한다.
		if modelName == "" {
			modelName = "MAP7700"
		}
		//fmt.Println(modelName,macarray[i],"####################################")
		if saveFg == "save" {
			fmt.Println("설정 오류")
		} else {
			//
			// 일단 모델무관 파일 존재 확인 및 압축 풀어 놓기
			_, err := ioutil.ReadFile("./file_storage/cf_" + macTag + ".zip") // 파일 유무 확인
			if err != nil {
				readDeault = "해당장비의 설정파일이 없습니다. 템플릿을 사용합니다 --- \"IP 주소를 꼭 변경해 주세요 !!\" (IP충돌주의)"
				fmt.Print("해당 장비 설정 파일이 없습니다. default 설정파일을 읽어 옵니다\n")
				if modelName == "MAP7500" {
					_, err = ioutil.ReadFile("./file_storage/cf_default7500.zip")
					if err != nil {
						print("cf_default7500.zip 파일도 없습니다. 시스템 점검이 필요합니다\n")
						os.Exit(7878)
					}
				} else if modelName == "MAP7700" || modelName == "MAP9220" || modelName == "MAP9200" || modelName == "MAP5010" || modelName == "MAP2010" { //장비추가
					_, err = ioutil.ReadFile("./file_storage/cf_default.zip")
					if err != nil {
						print("cf_default.zip 파일도 없습니다. 시스템 점검이 필요합니다\n")
						os.Exit(8888)
					}
				} else {
					println("\n모델명이 잘못되었습니다. MAC 주소 확인바랍니다..\n")
					w.Write([]byte("Wrong modelName. Needs Firmware Update"))
					return // 모델명 못찾으면 리턴해야 한다
				}
				noZipFileFlag = "DEFAULT"
			}
			// 어느 경우든지 macTag에 해당하는 파일 디렉토리 & 설정 파일들을 만든다.
			if noZipFileFlag == "DEFAULT" {
				// 디폴트 파일을 해당 MAC으로 풀어 놓는다.. 자가 증식 ~~
				if modelName == "MAP7500" {
					err = Unzip("./file_storage/cf_default7500.zip", "./file_storage/cf_"+macTag)
					if err != nil {
						fmt.Println(err)
					}
				} else if modelName == "MAP7700" || modelName == "MAP9220" || modelName == "MAP9200" || modelName == "MAP5010" || modelName == "MAP2010" { //장비추가
					err = Unzip("./file_storage/cf_default.zip", "./file_storage/cf_"+macTag)
					if err != nil {
						fmt.Println(err)
					}
				} else {
					println("\n모델명 분기 에러 입니다. MAC 주소 확인바랍니다..\n")
					return // 모델명 못찾으면 리턴해야 한다
				}
			} else {
				// 직전 zip으로 만든 파일을 풀어 놓는다.
				err = Unzip("./file_storage/cf_"+macTag+".zip", "./file_storage/cf_"+macTag)
				if err != nil {
					fmt.Println(err)
				}
			}

			// 모델에 따라 설정 슬라이스 가져오기
			if modelName == "MAP7500" {
				filePath0 := "./file_storage/cf_" + macTag + "/config" // 변경할 대상 파일 위치
				summaryData0 := map[string]string{"": ""}
				firmWareStr0 := "7500" // 파싱할 파일 스타일
				cf_matrix0 = updateConfigFile(filePath0, summaryData0, firmWareStr0)

				filePath1 := "./file_storage/cf_" + macTag + "/config.rf0" // 변경할 대상 파일 위치
				summaryData1 := map[string]string{"": ""}
				firmWareStr1 := "7500" // 파싱할 파일 스타일
				cf_matrix1 = updateConfigFile(filePath1, summaryData1, firmWareStr1)

				filePath2 := "./file_storage/cf_" + macTag + "/config.rf1" // 변경할 대상 파일 위치
				summaryData2 := map[string]string{"": ""}
				firmWareStr2 := "7500" // 파싱할 파일 스타일
				cf_matrix2 = updateConfigFile(filePath2, summaryData2, firmWareStr2)

				for i := 0; i < len(cf_matrix0); i++ {
					cf_map0[cf_matrix0[i][0]] = cf_matrix0[i][1]
					config_string += cf_matrix0[i][0] + " : " + cf_matrix0[i][1] + "\n"
				}
				config_string += "\n#####################################\n"
				for i := 0; i < len(cf_matrix1); i++ {
					cf_map1[cf_matrix1[i][0]] = cf_matrix1[i][1]
					config_string += cf_matrix1[i][0] + " : " + cf_matrix1[i][1] + "\n"
				}
				config_string += "\n#####################################\n"
				for i := 0; i < len(cf_matrix2); i++ {
					cf_map2[cf_matrix2[i][0]] = cf_matrix2[i][1]
					config_string += cf_matrix2[i][0] + " : " + cf_matrix2[i][1] + "\n"
				}

			} else if modelName == "MAP7700" || modelName == "MAP9220" || modelName == "MAP9200" || modelName == "MAP5010" || modelName == "MAP2010" { //장비추가
				filePath0 := "./file_storage/cf_" + macTag + "/RT2880_Settings.dat" // 변경할 대상 파일 위치
				summaryData0 := map[string]string{"": ""}
				firmWareStr0 := "7700_RT2880_Settings" // 파싱할 파일 스타일
				cf_matrix0 = updateConfigFile(filePath0, summaryData0, firmWareStr0)

				filePath1 := "./file_storage/cf_" + macTag + "/iNIC_ap.dat" // 변경할 대상 파일 위치
				summaryData1 := map[string]string{"": ""}
				firmWareStr1 := "7700_iNIC_ap" // 파싱할 파일 스타일
				cf_matrix1 = updateConfigFile(filePath1, summaryData1, firmWareStr1)

				for i := 0; i < len(cf_matrix0); i++ {
					cf_map0[cf_matrix0[i][0]] = cf_matrix0[i][1]
					config_string += cf_matrix0[i][0] + " : " + cf_matrix0[i][1] + "\n"
				}
				config_string += "\n#####################################\n"
				for i := 0; i < len(cf_matrix1); i++ {
					cf_map1[cf_matrix1[i][0]] = cf_matrix1[i][1]
					config_string += cf_matrix1[i][0] + " : " + cf_matrix1[i][1] + "\n"
				}
			}

			{

				switch modelName {
				case "MAP7500":
					if i == 0 {
						mac11 = macAddr
						ip11 = cf_map0["IP_ADDRESS"]
						id11 = cf_map1["ESSID[0]"]
						pass11 = cf_map1["WPAP_PASSPHRASE[0]"]
						id5g11 = cf_map2["ESSID[1]"]
						pass5g11 = cf_map2["WPAP_PASSPHRASE[1]"]
						gateway = cf_map0["DEFAULT_GATEWAY"]
						curl = cf_map0["CURL_SERVER_URL"]
					}
					fmt.Println(curl)
					if i == 1 {
						mac22 = macAddr
						ip22 = cf_map0["IP_ADDRESS"]
						id22 = cf_map1["ESSID[0]"]
						pass22 = cf_map1["WPAP_PASSPHRASE[0]"]
						id5g22 = cf_map2["ESSID[1]"]
						pass5g22 = cf_map2["WPAP_PASSPHRASE[1]"]

					}
					if i == 2 {
						mac33 = macAddr
						ip33 = cf_map0["IP_ADDRESS"]
						id33 = cf_map1["ESSID[0]"]
						pass33 = cf_map1["WPAP_PASSPHRASE[0]"]
						id5g33 = cf_map2["ESSID[1]"]
						pass5g33 = cf_map2["WPAP_PASSPHRASE[1]"]

					}
					if i == 3 {
						mac44 = macAddr
						ip44 = cf_map0["IP_ADDRESS"]
						id44 = cf_map1["ESSID[0]"]
						pass44 = cf_map1["WPAP_PASSPHRASE[0]"]
						id5g44 = cf_map2["ESSID[1]"]
						pass5g44 = cf_map2["WPAP_PASSPHRASE[1]"]

					}
					if i == 4 {
						mac55 = macAddr
						ip55 = cf_map0["IP_ADDRESS"]
						id55 = cf_map1["ESSID[0]"]
						pass55 = cf_map1["WPAP_PASSPHRASE[0]"]
						id5g55 = cf_map2["ESSID[1]"]
						pass5g55 = cf_map2["WPAP_PASSPHRASE[1]"]

					}
					if i == 5 {
						mac66 = macAddr
						ip66 = cf_map0["IP_ADDRESS"]
						id66 = cf_map1["ESSID[0]"]
						pass66 = cf_map1["WPAP_PASSPHRASE[0]"]
						id5g66 = cf_map2["ESSID[1]"]
						pass5g66 = cf_map2["WPAP_PASSPHRASE[1]"]

					}
					if i == 6 {
						mac77 = macAddr
						ip77 = cf_map0["IP_ADDRESS"]
						id77 = cf_map1["ESSID[0]"]
						pass77 = cf_map1["WPAP_PASSPHRASE[0]"]
						id5g77 = cf_map2["ESSID[1]"]
						pass5g77 = cf_map2["WPAP_PASSPHRASE[1]"]

					}
					if i == 7 {
						mac88 = macAddr
						ip88 = cf_map0["IP_ADDRESS"]
						id88 = cf_map1["ESSID[0]"]
						pass88 = cf_map1["WPAP_PASSPHRASE[0]"]
						id5g88 = cf_map2["ESSID[1]"]
						pass5g88 = cf_map2["WPAP_PASSPHRASE[1]"]

					}
					if i == 8 {
						mac99 = macAddr
						ip99 = cf_map0["IP_ADDRESS"]
						id99 = cf_map1["ESSID[0]"]
						pass99 = cf_map1["WPAP_PASSPHRASE[0]"]
						id5g99 = cf_map2["ESSID[1]"]
						pass5g99 = cf_map2["WPAP_PASSPHRASE[1]"]

					}
					if i == 9 {
						mac10 = macAddr
						ip10 = cf_map0["IP_ADDRESS"]
						id10 = cf_map1["ESSID[0]"]
						pass10 = cf_map1["WPAP_PASSPHRASE[0]"]
						id5g10 = cf_map2["ESSID[1]"]
						pass5g10 = cf_map2["WPAP_PASSPHRASE[1]"]

					}

				case "MAP7700":
					if i == 0 {
						ip11 = cf_map0["lan_ipaddr"]
						id11 = cf_map0["SSID1"]
						mac11 = macAddr
						pass11 = cf_map0["WPAPSK1"]
						id5g11 = cf_map1["SSID1"]
						pass5g11 = cf_map1["WPAPSK1"]
						gateway = cf_map0["lan_gateway"]
						curl = cf_map0["curl_server_url"]
					}
					if i == 1 {
						ip22 = cf_map0["lan_ipaddr"]
						id22 = cf_map0["SSID1"]
						mac22 = macAddr
						pass22 = cf_map0["WPAPSK1"]
						id5g22 = cf_map1["SSID1"]
						pass5g22 = cf_map1["WPAPSK1"]
					}
					if i == 2 {
						ip33 = cf_map0["lan_ipaddr"]
						id33 = cf_map0["SSID1"]
						mac33 = macAddr
						pass33 = cf_map0["WPAPSK1"]
						id5g33 = cf_map1["SSID1"]
						pass5g33 = cf_map1["WPAPSK1"]
					}
					if i == 3 {
						ip44 = cf_map0["lan_ipaddr"]
						id44 = cf_map0["SSID1"]
						mac44 = macAddr
						pass44 = cf_map0["WPAPSK1"]
						id5g44 = cf_map1["SSID1"]
						pass5g44 = cf_map1["WPAPSK1"]
					}
					if i == 4 {
						ip55 = cf_map0["lan_ipaddr"]
						id55 = cf_map0["SSID1"]
						mac55 = macAddr
						pass55 = cf_map0["WPAPSK1"]
						id5g55 = cf_map1["SSID1"]
						pass5g55 = cf_map1["WPAPSK1"]
					}
					if i == 5 {
						ip66 = cf_map0["lan_ipaddr"]
						id66 = cf_map0["SSID1"]
						mac66 = macAddr
						pass66 = cf_map0["WPAPSK1"]
						id5g66 = cf_map1["SSID1"]
						pass5g66 = cf_map1["WPAPSK1"]
					}
					if i == 6 {
						ip77 = cf_map0["lan_ipaddr"]
						id77 = cf_map0["SSID1"]
						mac77 = macAddr
						pass77 = cf_map0["WPAPSK1"]
						id5g77 = cf_map1["SSID1"]
						pass5g77 = cf_map1["WPAPSK1"]
					}
					if i == 7 {
						ip88 = cf_map0["lan_ipaddr"]
						id88 = cf_map0["SSID1"]
						mac88 = macAddr
						pass88 = cf_map0["WPAPSK1"]
						id5g88 = cf_map1["SSID1"]
						pass5g88 = cf_map1["WPAPSK1"]
					}
					if i == 8 {
						ip99 = cf_map0["lan_ipaddr"]
						id99 = cf_map0["SSID1"]
						mac99 = macAddr
						pass99 = cf_map0["WPAPSK1"]
						id5g99 = cf_map1["SSID1"]
						pass5g99 = cf_map1["WPAPSK1"]
					}
					if i == 9 {
						ip10 = cf_map0["lan_ipaddr"]
						id10 = cf_map0["SSID1"]
						mac10 = macAddr
						pass10 = cf_map0["WPAPSK1"]
						id5g10 = cf_map1["SSID1"]
						pass5g10 = cf_map1["WPAPSK1"]
					}
				case "MAP9220":
					if i == 0 {
						ip11 = cf_map0["lan_ipaddr"]
						id11 = cf_map0["SSID1"]
						mac11 = macAddr
						pass11 = cf_map0["WPAPSK1"]
						id5g11 = cf_map1["SSID1"]
						pass5g11 = cf_map1["WPAPSK1"]
						gateway = cf_map0["lan_gateway"]
						curl = cf_map0["curl_server_url"]
					}
					if i == 1 {
						ip22 = cf_map0["lan_ipaddr"]
						id22 = cf_map0["SSID1"]
						mac22 = macAddr
						pass22 = cf_map0["WPAPSK1"]
						id5g22 = cf_map1["SSID1"]
						pass5g22 = cf_map1["WPAPSK1"]
					}
					if i == 2 {
						ip33 = cf_map0["lan_ipaddr"]
						id33 = cf_map0["SSID1"]
						mac33 = macAddr
						pass33 = cf_map0["WPAPSK1"]
						id5g33 = cf_map1["SSID1"]
						pass5g33 = cf_map1["WPAPSK1"]
					}
					if i == 3 {
						ip44 = cf_map0["lan_ipaddr"]
						id44 = cf_map0["SSID1"]
						mac44 = macAddr
						pass44 = cf_map0["WPAPSK1"]
						id5g44 = cf_map1["SSID1"]
						pass5g44 = cf_map1["WPAPSK1"]
					}
					if i == 4 {
						ip55 = cf_map0["lan_ipaddr"]
						id55 = cf_map0["SSID1"]
						mac55 = macAddr
						pass55 = cf_map0["WPAPSK1"]
						id5g55 = cf_map1["SSID1"]
						pass5g55 = cf_map1["WPAPSK1"]
					}
					if i == 5 {
						ip66 = cf_map0["lan_ipaddr"]
						id66 = cf_map0["SSID1"]
						mac66 = macAddr
						pass66 = cf_map0["WPAPSK1"]
						id5g66 = cf_map1["SSID1"]
						pass5g66 = cf_map1["WPAPSK1"]
					}
					if i == 6 {
						ip77 = cf_map0["lan_ipaddr"]
						id77 = cf_map0["SSID1"]
						mac77 = macAddr
						pass77 = cf_map0["WPAPSK1"]
						id5g77 = cf_map1["SSID1"]
						pass5g77 = cf_map1["WPAPSK1"]
					}
					if i == 7 {
						ip88 = cf_map0["lan_ipaddr"]
						id88 = cf_map0["SSID1"]
						mac88 = macAddr
						pass88 = cf_map0["WPAPSK1"]
						id5g88 = cf_map1["SSID1"]
						pass5g88 = cf_map1["WPAPSK1"]
					}
					if i == 8 {
						ip99 = cf_map0["lan_ipaddr"]
						id99 = cf_map0["SSID1"]
						mac99 = macAddr
						pass99 = cf_map0["WPAPSK1"]
						id5g99 = cf_map1["SSID1"]
						pass5g99 = cf_map1["WPAPSK1"]
					}
					if i == 9 {
						ip10 = cf_map0["lan_ipaddr"]
						id10 = cf_map0["SSID1"]
						mac10 = macAddr
						pass10 = cf_map0["WPAPSK1"]
						id5g10 = cf_map1["SSID1"]
						pass5g10 = cf_map1["WPAPSK1"]
					}
				case "MAP9200":
					if i == 0 {
						ip11 = cf_map0["lan_ipaddr"]
						id11 = cf_map0["SSID1"]
						mac11 = macAddr
						pass11 = cf_map0["WPAPSK1"]
						id5g11 = cf_map1["SSID1"]
						pass5g11 = cf_map1["WPAPSK1"]
						gateway = cf_map0["lan_gateway"]
						curl = cf_map0["curl_server_url"]
					}
					if i == 1 {
						ip22 = cf_map0["lan_ipaddr"]
						id22 = cf_map0["SSID1"]
						mac22 = macAddr
						pass22 = cf_map0["WPAPSK1"]
						id5g22 = cf_map1["SSID1"]
						pass5g22 = cf_map1["WPAPSK1"]
					}
					if i == 2 {
						ip33 = cf_map0["lan_ipaddr"]
						id33 = cf_map0["SSID1"]
						mac33 = macAddr
						pass33 = cf_map0["WPAPSK1"]
						id5g33 = cf_map1["SSID1"]
						pass5g33 = cf_map1["WPAPSK1"]
					}
					if i == 3 {
						ip44 = cf_map0["lan_ipaddr"]
						id44 = cf_map0["SSID1"]
						mac44 = macAddr
						pass44 = cf_map0["WPAPSK1"]
						id5g44 = cf_map1["SSID1"]
						pass5g44 = cf_map1["WPAPSK1"]
					}
					if i == 4 {
						ip55 = cf_map0["lan_ipaddr"]
						id55 = cf_map0["SSID1"]
						mac55 = macAddr
						pass55 = cf_map0["WPAPSK1"]
						id5g55 = cf_map1["SSID1"]
						pass5g55 = cf_map1["WPAPSK1"]
					}
					if i == 5 {
						ip66 = cf_map0["lan_ipaddr"]
						id66 = cf_map0["SSID1"]
						mac66 = macAddr
						pass66 = cf_map0["WPAPSK1"]
						id5g66 = cf_map1["SSID1"]
						pass5g66 = cf_map1["WPAPSK1"]
					}
					if i == 6 {
						ip77 = cf_map0["lan_ipaddr"]
						id77 = cf_map0["SSID1"]
						mac77 = macAddr
						pass77 = cf_map0["WPAPSK1"]
						id5g77 = cf_map1["SSID1"]
						pass5g77 = cf_map1["WPAPSK1"]
					}
					if i == 7 {
						ip88 = cf_map0["lan_ipaddr"]
						id88 = cf_map0["SSID1"]
						mac88 = macAddr
						pass88 = cf_map0["WPAPSK1"]
						id5g88 = cf_map1["SSID1"]
						pass5g88 = cf_map1["WPAPSK1"]
					}
					if i == 8 {
						ip99 = cf_map0["lan_ipaddr"]
						id99 = cf_map0["SSID1"]
						mac99 = macAddr
						pass99 = cf_map0["WPAPSK1"]
						id5g99 = cf_map1["SSID1"]
						pass5g99 = cf_map1["WPAPSK1"]
					}
					if i == 9 {
						ip10 = cf_map0["lan_ipaddr"]
						id10 = cf_map0["SSID1"]
						mac10 = macAddr
						pass10 = cf_map0["WPAPSK1"]
						id5g10 = cf_map1["SSID1"]
						pass5g10 = cf_map1["WPAPSK1"]
					}
				case "MAP5010":
					if i == 0 {
						ip11 = cf_map0["lan_ipaddr"]
						id11 = cf_map0["SSID1"]
						mac11 = macAddr
						pass11 = cf_map0["WPAPSK1"]
						id5g11 = cf_map1["SSID1"]
						pass5g11 = cf_map1["WPAPSK1"]
						gateway = cf_map0["lan_gateway"]
						curl = cf_map0["curl_server_url"]
					}
					if i == 1 {
						ip22 = cf_map0["lan_ipaddr"]
						id22 = cf_map0["SSID1"]
						mac22 = macAddr
						pass22 = cf_map0["WPAPSK1"]
						id5g22 = cf_map1["SSID1"]
						pass5g22 = cf_map1["WPAPSK1"]
					}
					if i == 2 {
						ip33 = cf_map0["lan_ipaddr"]
						id33 = cf_map0["SSID1"]
						mac33 = macAddr
						pass33 = cf_map0["WPAPSK1"]
						id5g33 = cf_map1["SSID1"]
						pass5g33 = cf_map1["WPAPSK1"]
					}
					if i == 3 {
						ip44 = cf_map0["lan_ipaddr"]
						id44 = cf_map0["SSID1"]
						mac44 = macAddr
						pass44 = cf_map0["WPAPSK1"]
						id5g44 = cf_map1["SSID1"]
						pass5g44 = cf_map1["WPAPSK1"]
					}
					if i == 4 {
						ip55 = cf_map0["lan_ipaddr"]
						id55 = cf_map0["SSID1"]
						mac55 = macAddr
						pass55 = cf_map0["WPAPSK1"]
						id5g55 = cf_map1["SSID1"]
						pass5g55 = cf_map1["WPAPSK1"]
					}
					if i == 5 {
						ip66 = cf_map0["lan_ipaddr"]
						id66 = cf_map0["SSID1"]
						mac66 = macAddr
						pass66 = cf_map0["WPAPSK1"]
						id5g66 = cf_map1["SSID1"]
						pass5g66 = cf_map1["WPAPSK1"]
					}
					if i == 6 {
						ip77 = cf_map0["lan_ipaddr"]
						id77 = cf_map0["SSID1"]
						mac77 = macAddr
						pass77 = cf_map0["WPAPSK1"]
						id5g77 = cf_map1["SSID1"]
						pass5g77 = cf_map1["WPAPSK1"]
					}
					if i == 7 {
						ip88 = cf_map0["lan_ipaddr"]
						id88 = cf_map0["SSID1"]
						mac88 = macAddr
						pass88 = cf_map0["WPAPSK1"]
						id5g88 = cf_map1["SSID1"]
						pass5g88 = cf_map1["WPAPSK1"]
					}
					if i == 8 {
						ip99 = cf_map0["lan_ipaddr"]
						id99 = cf_map0["SSID1"]
						mac99 = macAddr
						pass99 = cf_map0["WPAPSK1"]
						id5g99 = cf_map1["SSID1"]
						pass5g99 = cf_map1["WPAPSK1"]
					}
					if i == 9 {
						ip10 = cf_map0["lan_ipaddr"]
						id10 = cf_map0["SSID1"]
						mac10 = macAddr
						pass10 = cf_map0["WPAPSK1"]
						id5g10 = cf_map1["SSID1"]
						pass5g10 = cf_map1["WPAPSK1"]
					}
				case "MAP2010":
					if i == 0 {
						ip11 = cf_map0["lan_ipaddr"]
						id11 = cf_map0["SSID1"]
						mac11 = macAddr
						pass11 = cf_map0["WPAPSK1"]
						id5g11 = cf_map1["SSID1"]
						pass5g11 = cf_map1["WPAPSK1"]
						gateway = cf_map0["lan_gateway"]
						curl = cf_map0["curl_server_url"]
					}
					if i == 1 {
						ip22 = cf_map0["lan_ipaddr"]
						id22 = cf_map0["SSID1"]
						mac22 = macAddr
						pass22 = cf_map0["WPAPSK1"]
						id5g22 = cf_map1["SSID1"]
						pass5g22 = cf_map1["WPAPSK1"]
					}
					if i == 2 {
						ip33 = cf_map0["lan_ipaddr"]
						id33 = cf_map0["SSID1"]
						mac33 = macAddr
						pass33 = cf_map0["WPAPSK1"]
						id5g33 = cf_map1["SSID1"]
						pass5g33 = cf_map1["WPAPSK1"]
					}
					if i == 3 {
						ip44 = cf_map0["lan_ipaddr"]
						id44 = cf_map0["SSID1"]
						mac44 = macAddr
						pass44 = cf_map0["WPAPSK1"]
						id5g44 = cf_map1["SSID1"]
						pass5g44 = cf_map1["WPAPSK1"]
					}
					if i == 4 {
						ip55 = cf_map0["lan_ipaddr"]
						id55 = cf_map0["SSID1"]
						mac55 = macAddr
						pass55 = cf_map0["WPAPSK1"]
						id5g55 = cf_map1["SSID1"]
						pass5g55 = cf_map1["WPAPSK1"]
					}
					if i == 5 {
						ip66 = cf_map0["lan_ipaddr"]
						id66 = cf_map0["SSID1"]
						mac66 = macAddr
						pass66 = cf_map0["WPAPSK1"]
						id5g66 = cf_map1["SSID1"]
						pass5g66 = cf_map1["WPAPSK1"]
					}
					if i == 6 {
						ip77 = cf_map0["lan_ipaddr"]
						id77 = cf_map0["SSID1"]
						mac77 = macAddr
						pass77 = cf_map0["WPAPSK1"]
						id5g77 = cf_map1["SSID1"]
						pass5g77 = cf_map1["WPAPSK1"]
					}
					if i == 7 {
						ip88 = cf_map0["lan_ipaddr"]
						id88 = cf_map0["SSID1"]
						mac88 = macAddr
						pass88 = cf_map0["WPAPSK1"]
						id5g88 = cf_map1["SSID1"]
						pass5g88 = cf_map1["WPAPSK1"]
					}
					if i == 8 {
						ip99 = cf_map0["lan_ipaddr"]
						id99 = cf_map0["SSID1"]
						mac99 = macAddr
						pass99 = cf_map0["WPAPSK1"]
						id5g99 = cf_map1["SSID1"]
						pass5g99 = cf_map1["WPAPSK1"]
					}
					if i == 9 {
						ip10 = cf_map0["lan_ipaddr"]
						id10 = cf_map0["SSID1"]
						mac10 = macAddr
						pass10 = cf_map0["WPAPSK1"]
						id5g10 = cf_map1["SSID1"]
						pass5g10 = cf_map1["WPAPSK1"]
					}

				}
				conf_ := configStruct{
					ReadDefaultFile: readDeault,
					ModelName:       modelName,
					MacAddr1:        mac11,
					MacAddr2:        mac22,
					MacAddr3:        mac33,
					MacAddr4:        mac44,
					MacAddr5:        mac55,
					MacAddr6:        mac66,
					MacAddr7:        mac77,
					MacAddr8:        mac88,
					MacAddr9:        mac99,
					MacAddr10:       mac10,
					CHECKED:         checked,
					Lan_ipaddr1:     ip11,
					Lan_ipaddr2:     ip22,
					Lan_ipaddr3:     ip33,
					Lan_ipaddr4:     ip44,
					Lan_ipaddr5:     ip55,
					Lan_ipaddr6:     ip66,
					Lan_ipaddr7:     ip77,
					Lan_ipaddr8:     ip88,
					Lan_ipaddr9:     ip99,
					Lan_ipaddr10:    ip10, // 메인보드 온리
					Lan_gateway:     gateway,
					SSID_2G1:        id11,
					SSID_2G2:        id22,
					SSID_2G3:        id33,
					SSID_2G4:        id44,
					SSID_2G5:        id55,
					SSID_2G6:        id66,
					SSID_2G7:        id77,
					SSID_2G8:        id88,
					SSID_2G9:        id99,
					SSID_2G10:       id10,
					SSID_5G1:        id5g11,
					SSID_5G2:        id5g22,
					SSID_5G3:        id5g33,
					SSID_5G4:        id5g44,
					SSID_5G5:        id5g55,
					SSID_5G6:        id5g66,
					SSID_5G7:        id5g77,
					SSID_5G8:        id5g88,
					SSID_5G9:        id5g99,
					SSID_5G10:       id5g10,
					SSID_5G:         cf_map2["ESSID[1]"], // 무선 2
					WPAPSK_2G1:      pass11,
					WPAPSK_2G2:      pass22,
					WPAPSK_2G3:      pass33,
					WPAPSK_2G4:      pass44,
					WPAPSK_2G5:      pass55,
					WPAPSK_2G6:      pass66,
					WPAPSK_2G7:      pass77,
					WPAPSK_2G8:      pass88,
					WPAPSK_2G9:      pass99,
					WPAPSK_2G10:     pass10,
					WPAPSK_5G1:      pass5g11,
					WPAPSK_5G2:      pass5g22,
					WPAPSK_5G3:      pass5g33,
					WPAPSK_5G4:      pass5g44,
					WPAPSK_5G5:      pass5g55,
					WPAPSK_5G6:      pass5g66,
					WPAPSK_5G7:      pass5g77,
					WPAPSK_5G8:      pass5g88,
					WPAPSK_5G9:      pass5g99,
					WPAPSK_5G10:     pass5g10,
					CURL:            curl,
				}
				cf[0] = conf_
			}

		}
	}
	if noZipFileFlag == "DEFAULT" {
		w.Write([]byte("Push [Edit CF Files] to set IP first"))
	} else if modelName == "UnKnown" {

	} else {
		conf_template.Execute(w, cf[0])
	}

}

func saveConfig(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()
	db, err := sql.Open("mysql", "root:1111@tcp("+dbLocation+":"+db_port+")/map")
	if err != nil {
		panic(err)
	}
	//st = "site" + curYearMonth // ""사이트"" 테이블명 확정
	//db.Exec("TRUNCATE TABLE "+st)
	defer db.Close()

	//fmt.Println(r)
	var dat string
	if dateRange, ok := r.Form["macAddr"]; !ok {
		fmt.Println("대상 AP MAC 주소가 없습니다.. 사용법 확인하세요", dateRange)
		fmt.Printf("%T  --------\n", dat)
		w.Write([]byte("쿼리에 AP MAC 주소가 없습니다.. 사용법 확인하세요"))
		return
	}
	// macTag 확정
	var macTag string
	var modelName string
	var conf_table string

	saveFlag := r.Form.Get("saveFlag")
	saveFg := "" + saveFlag

	mac := r.Form.Get("macAddr")
	println("------------------->>>>", mac)
	macAddr := "" + mac
	mac = strings.Replace(mac, ":", "", -1)
	macTag = mac[len(mac)-6:]
	curYearMonth := time.Now().String()
	curYearBuf := curYearMonth[:4]                // 연도 짤라내기. 글자수는 0부터 시작하고 끝 숫자는 빠진다. 0,1,2,3 위치 잘라내기
	curYearMonth = curYearBuf + curYearMonth[5:7] // 월 짤라내기
	site := "site" + curYearMonth
	ipQuerySt1 := "SELECT modelName from " + site + " WHERE macAddr = " + "\"" + macAddr + "\"" /// 물어볼땐 기존 n 구조체 MAC을 사용 북마크북마크
	//fmt.Println("///////////111",ipQuerySt1)
	err = db.QueryRow(ipQuerySt1).Scan(&modelName) // s 구조체변수로 받아낸다
	cf_matrix0 := [][]string{}                     //변수길이를 못쓰니까 slice로 대체
	cf_matrix1 := [][]string{}                     //변수길이를 못쓰니까 slice로 대체
	cf_matrix2 := [][]string{}                     //변수길이를 못쓰니까 slice로 대체 (7500 때문에 총 3개)
	cf_map0 := map[string]string{}
	cf_map1 := map[string]string{}
	cf_map2 := map[string]string{}
	if modelName == "" {
		modelName = "MAP7700"
	}
	if saveFg == "save" {
		if modelName == "MAP7500" {
			cc1 := map[string]string{ // 첫번째 파일 수정용 map
				"DEFAULT_GATEWAY": "",
				"CURL_SERVER_URL": "",
			}
			cc2 := map[string]string{ // 2번째 파일 수정용 map
				"ESSID[0]":           "",
				"WPAP_PASSPHRASE[0]": "",
				"SECURITY_MODE[0]":   "",
			}
			cc3 := map[string]string{ // 3번째 파일 수정용 map
				"ESSID[1]":           "",
				"WPAP_PASSPHRASE[1]": "",
				"SECURITY_MODE[1]":   "",
			}
			for k, v := range r.Form {
				tempV := strings.Join(v, "") // 문자 map value 값을 string으로 변환하기
				switch k {
				case "Lan_gateway":
					cc1["DEFAULT_GATEWAY"] = tempV
				case "SSID_2G":
					cc2["ESSID[0]"] = tempV
				case "WPAPSK_2G":
					cc2["WPAP_PASSPHRASE[0]"] = tempV
				case "SSID_5G":
					cc3["ESSID[1]"] = tempV
				case "WPAPSK_5G":
					cc3["WPAP_PASSPHRASE[1]"] = tempV
				case "Curlserver":
					cc1["CURL_SERVER_URL"] = tempV
				default:
				}

			}
			if len(cc2["WPAP_PASSPHRASE[0]"]) == 0 {
				cc2["SECURITY_MODE[0]"] = "DISABLE"
			} else {
				cc2["SECURITY_MODE[0]"] = "WPA2_PERSONAL"
			}

			if len(cc3["WPAP_PASSPHRASE[1]"]) == 0 {
				cc3["SECURITY_MODE[1]"] = "DISABLE"
			} else {
				cc3["SECURITY_MODE[1]"] = "WPA2_PERSONAL"
			}

			filePath0 := "./file_storage/cf_" + macTag + "/config" // 변경할 대상 파일 위치
			summaryData0 := cc1                                    // 변경할 내용
			firmWareStr0 := "7500"                                 // 파싱할 파일 스타일
			cf_matrix0 = updateConfigFile(filePath0, summaryData0, firmWareStr0)

			filePath1 := "./file_storage/cf_" + macTag + "/config.rf0" // 변경할 대상 파일 위치
			summaryData1 := cc2                                        // 변경할 내용
			firmWareStr1 := "7500"                                     // 파싱할 파일 스타일
			cf_matrix1 = updateConfigFile(filePath1, summaryData1, firmWareStr1)

			filePath2 := "./file_storage/cf_" + macTag + "/config.rf1" // 변경할 대상 파일 위치
			summaryData2 := cc3                                        // 변경할 내용
			firmWareStr2 := "7500"                                     // 파싱할 파일 스타일
			cf_matrix2 = updateConfigFile(filePath2, summaryData2, firmWareStr2)
			//fmt.Println(cf_matrix0, cf_matrix1)

			// 해당 mac 값의 파일
			//ArchiveFile("./file_storage/cf_" + macTag + "/", "./file_storage/cf_" + macTag + ".zip", nil )
			go func() {
				if my7zip(macTag) != nil {
					// 혹시 외부함수로 서버 기능들 블럭 될까봐...
					print("압축 오류 발생 !!")
				}
			}()

			textMsg0 := ""
			textMsg1 := ""
			textMsg2 := ""

			for i := 0; i < len(cf_matrix0); i++ {
				cf_map0[cf_matrix0[i][0]] = cf_matrix0[i][1]
				textMsg0 += cf_matrix0[i][0] + " : " + cf_matrix0[i][1] + "\n"
			}
			for i := 0; i < len(cf_matrix1); i++ {
				cf_map1[cf_matrix1[i][0]] = cf_matrix1[i][1]
				textMsg1 += cf_matrix1[i][0] + " : " + cf_matrix1[i][1] + "\n"
			}

			for i := 0; i < len(cf_matrix2); i++ {
				cf_map2[cf_matrix2[i][0]] = cf_matrix2[i][1]
				textMsg2 += cf_matrix2[i][0] + " : " + cf_matrix2[i][1] + "\n"
			}

			w.Header().Set("Content-Type", "text/html")
			conf_table =
				"<!DOCTYPE html> <html>	<html lang=\"en\"> <head> <style> table {width:100%;} table, th, td {" + " border: 1px solid black; border-collapse: collapse;	} th, td { padding: 5px; text-align: center;" + "} table.names tr:nth-child(even) { background-color: #eee; }	table.names tr:nth-child(odd) {" + "background-color:#fff; } table.names th { background-color: black; color: yellow } </style>" + "<meta charset=\"UTF-8\"> </head> <body> <p>JMP SYSTEMS Co., Ltd. 2017</p><p></p><p></p><p></p><table class=\"names\"> <tr> <th>Zip 저장완료 </th>" + "<th>MAC: " + macAddr + "</th> </tr>"
			conf_table += "</table><div><p></p>ctl+w 또는 닫기 클릭하세요 <button onclick=\"self.close()\"> 닫기 </button><p></p></div>"
			conf_table += "<p>메인보드 유선설정</p>"
			conf_table += "<textarea readonly=\"readonly\" onclick=\"this.select()\" style= \"background-color:#f9e0fF; min-width:100%; max-width:100%; min-height:256px; max-height:256px\" >" + textMsg0 + "</textarea></br>"
			conf_table += "<p>무선1</p>"
			conf_table += "<textarea readonly=\"readonly\" onclick=\"this.select()\" style= \"background-color:#e9f5fF; min-width:100%; max-width:100%; min-height:256px; max-height:256px\" >" + textMsg1 + "</textarea></br>"
			conf_table += "<p>무선2</p>"
			conf_table += "<textarea readonly=\"readonly\" onclick=\"this.select()\" style= \"background-color:#e9e5ff; min-width:100%; max-width:100%; min-height:256px; max-height:256px\" >" + textMsg2 + "</textarea></br>"
			conf_table += "</body> </html>"

			w.Write([]byte(conf_table))
			w.Write([]byte(macAddr + " 장비의 설정파일이 zip으로 저장되었습니다\r\n"))

		} else if modelName == "MAP7700" || modelName == "MAP9220" || modelName == "MAP9200" || modelName == "MAP5010" || modelName == "MAP2010" { //2G 장비추가
			cc1 := map[string]string{
				"lan_gateway":     "",
				"SSID1":           "",
				"WPAPSK1":         "",
				"curl_server_url": "", ////////////추가//////////////
				"AuthMode":        "",
				"EncrypType":      "",
			}
			cc2 := map[string]string{ //5G
				"SSID1":      "",
				"WPAPSK1":    "",
				"AuthMode":   "",
				"EncrypType": "",
			}
			for k, v := range r.Form {
				tempV := strings.Join(v, "") // 문자 map value 값을 string으로 변환하기
				switch k {
				case "Lan_gateway":
					cc1["lan_gateway"] = tempV
				case "SSID_2G":
					cc1["SSID1"] = tempV
				case "WPAPSK_2G":
					cc1["WPAPSK1"] = tempV
				case "SSID_5G":
					cc2["SSID1"] = tempV
				case "WPAPSK_5G":
					cc2["WPAPSK1"] = tempV
				case "Secure_Mode_2G":
					cc1["AuthMode"] = tempV
				case "Secure_Mode_enc_2G":
					cc1["EncrypType"] = tempV
				case "Secure_Mode_5G":
					cc2["AuthMode"] = tempV
				case "Secure_Mode_enc_5G":
					cc2["EncrypType"] = tempV
				case "Curlserver": ////////////추가//////////////
					cc1["curl_server_url"] = tempV ////////////추가//////////////
				default:
				}
			}
		//	fmt.Println(len(cc1["WPAPSK1"]), cc1["WPAPSK1"], "////////")
			if len(cc1["WPAPSK1"]) < 8 {
				cc1["AuthMode"] = "OPEN"
				cc1["EncrypType"] = "NONE"
			} else {
				cc1["AuthMode"] = "WPA2PSK"
				cc1["EncrypType"] = "TKIPAES"
			}

			if len(cc2["WPAPSK1"]) < 8 {
				cc2["AuthMode"] = "OPEN"
				cc2["EncrypType"] = "NONE"
			} else {
				cc2["AuthMode"] = "WPA2PSK"
				cc2["EncrypType"] = "TKIPAES+"
			}
			//fmt.Println("+++++++++++++++++++", cc1,cc2)
			filePath0 := "./file_storage/cf_" + macTag + "/RT2880_Settings.dat" // 변경할 대상 파일 위치
			summaryData0 := cc1                                                 // 변경할 내용
			firmWareStr0 := "7700_RT2880_Settings"                              // 파싱할 파일 스타일
			cf_matrix0 = updateConfigFile(filePath0, summaryData0, firmWareStr0)

			filePath1 := "./file_storage/cf_" + macTag + "/iNIC_ap.dat" // 변경할 대상 파일 위치
			summaryData1 := cc2                                         // 변경할 내용
			firmWareStr1 := "7700_iNIC_ap"                              // 파싱할 파일 스타일
			cf_matrix1 = updateConfigFile(filePath1, summaryData1, firmWareStr1)

			//fmt.Println(cf_matrix0, cf_matrix1)

			// 해당 mac 값의 파일
			//ArchiveFile("./file_storage/cf_" + macTag + "/", "./file_storage/cf_" + macTag + ".zip", nil )
			go func() {
				if my7zip(macTag) != nil {
					// 혹시 외부함수로 서버 기능들 블럭 될까봐...
					print("압축 오류 발생 !!")
				}
			}()

			textMsg0 := ""
			textMsg1 := ""

			for i := 0; i < len(cf_matrix0); i++ {
				cf_map0[cf_matrix0[i][0]] = cf_matrix0[i][1]
				textMsg0 += cf_matrix0[i][0] + " : " + cf_matrix0[i][1] + "\n"
			}
			for i := 0; i < len(cf_matrix1); i++ {
				cf_map1[cf_matrix1[i][0]] = cf_matrix1[i][1]
				textMsg1 += cf_matrix1[i][0] + " : " + cf_matrix1[i][1] + "\n"
			}

			w.Header().Set("Content-Type", "text/html")
			conf_table =
				"<!DOCTYPE html> <html>	<html lang=\"en\"> <head> <style> table {width:100%;} table, th, td {" + " border: 1px solid black; border-collapse: collapse;	} th, td { padding: 5px; text-align: center;" + "} table.names tr:nth-child(even) { background-color: #eee; }	table.names tr:nth-child(odd) {" + "background-color:#fff; } table.names th { background-color: black; color: yellow } </style>" + "<meta charset=\"UTF-8\"> </head> <body> <p>JMP SYSTEMS Co., Ltd. 2017</p><p></p><p></p><p></p><table class=\"names\"> <tr> <th>Zip 저장완료 </th>" + "<th>MAC: " + macAddr + "</th> </tr>"
			conf_table += "</table><div><p></p>ctl+w 또는 닫기 클릭하세요 <button onclick=\"self.close()\"> 닫기 </button><p></p></div>"
			conf_table += "<p>RT2880Setting.dat</p>"
			conf_table += "<textarea readonly=\"readonly\" onclick=\"this.select()\" style= \"background-color:#D9E5FF; min-width:100%; max-width:100%; min-height:256px; max-height:256px\" >" + textMsg0 + "</textarea></br>"
			conf_table += "<p>iNIC_ap.dat</p>"
			conf_table += "<textarea readonly=\"readonly\" onclick=\"this.select()\" style= \"background-color:#e9f5fF; min-width:100%; max-width:100%; min-height:256px; max-height:256px\" >" + textMsg1 + "</textarea></br>"
			conf_table += "</body> </html>"

			w.Write([]byte(conf_table))
			w.Write([]byte(macAddr + " 장비의 설정파일이 zip으로 저장되었습니다\r\n"))
		}
	} else {
		fmt.Println("Savaflag가 제대로 오지 않음")
	}
}
func showData(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var dat string
	if dateRange, ok := r.Form["yearmonth"]; !ok {
		fmt.Println("연월이 없습니다.. 사용법 확인하세요 - [젬피]", dateRange)
		fmt.Printf("%T  --------\n", dat)
		w.Write([]byte("연월이 없습니다.. 사용법 확인하세요 - [젬피]"))
		return
	}
	var st string
	st = r.Form.Get("yearmonth")
	iNum, err := strconv.Atoi(st)
	// 형식 검사
	if err != nil {
		pp("쿼리 날짜 변환에 실패하였습니다")
		w.Write([]byte("쿼리 날짜 변환에 실패하였습니다"))
		return
	}
	// 자릿수 검사
	if len(st) != 6 {
		pp("연월 자릿수 오류 입니다.")
		w.Write([]byte("연월 자릿수 오류 입니다"))
		return
	}
	// 범위 검사  :::  13월 14월등은 검사하지 않음 !!!!!!!!  td-1612
	if (iNum > 201812) || (iNum < 201612) {
		pp("연월 지정 범위를 초과하였습니다")
		w.Write([]byte("연월 지정 범위를 초과하였습니다"))
		return
	}

	st = "ap" + st // 테이블 이름

	startTime := time.Now() // <---------------------------------- 스타트

	//db, err := sql.Open("mysql", "root:1111@tcp(www.umanager.kr:3306)/map")
	db, err := sql.Open("mysql", "root:1111@tcp("+dbLocation+":"+db_port+")/map") // www.umanager.kr:3306
	if err != nil {
		panic(err)
	}
	defer db.Close()

	var rowCnt int
	/* disk & cpu 부하줄이기 운동
	err = db.QueryRow("SELECT COUNT(*) FROM " + st + " WHERE id > ?", 0).Scan(&rowCnt)
	if err != nil {
		fmt.Println("[ERROR] 해당하는 테이블이 없습니다.", "테이블 이름 : ", st)
		w.Write([]byte("[ERROR] 해당하는 테이블이 없습니다.\r\n"))
		return
	}*/

	w.Write([]byte("\r\n테이블 열 갯수 : " + strconv.Itoa(rowCnt) + "\r\n\r\n")) // good

	rows, err := db.Query("select * from " + st)
	if err != nil {
		fmt.Println("db.Query 에서 에러발생--[JMP]")
		log.Fatal(err)
	}
	defer rows.Close()

	n := new(apTableVal)

	for rows.Next() {
		err := rows.Scan(&n.id, &n.time, &n.bootFlag, &n.macAddr, &n.ipAddr, &n.btVer, &n.fwVer, &n.ssidW0, &n.ssidW1, &n.chanW0, &n.chanW1,
			&n.rxByteW0, &n.txByteW0, &n.rxByteW1, &n.txByteW1, &n.rxPktW0, &n.txPktW0, &n.rxPktW1, &n.txPktW1, &n.assoCtW0,
			&n.assoCtW1, &n.devTemp, &n.curMem, &n.pingTime, &n.assoDevT, &n.leaveDevT) //, &n.tsuccess
		if err != nil {
			log.Fatal(err)
		}

			screenShot := fmt.Sprint(n.id, ",", n.time, ",", n.bootFlag, ",", n.macAddr, ",", n.ipAddr, ",", n.btVer, ",", n.fwVer, ",", n.ssidW0, ",", n.ssidW1, ",",
				n.chanW0, ",", n.chanW1, ",", n.rxByteW0, ",", n.txByteW0, ",", n.rxByteW1, ",", n.txByteW1, ",", n.rxPktW0, ",", n.txPktW0, ",", n.rxPktW1, ",", n.txPktW1, ",", n.assoCtW0, ",",
				n.assoCtW1, ",", n.devTemp, ",", n.curMem, ",", n.pingTime, ",", n.assoDevT, ",", n.leaveDevT, ",", n.peerAp0, ",", n.peerAp1, ",", n.staMac0) //,",",n.tsuccess
			w.Write([]byte(screenShot + "\r\n"))



	}

	elapsedTime := time.Since(startTime) // <---------------------- 엔드
	fmt.Printf("\n실행시간 : [%s]  ", elapsedTime)

	fmt.Println("\n\n", st, " 테이블 자료 출력이 완료되었습니다.\n")
	w.Write([]byte("\n\n" + st + " 테이블 자료 출력이 완료되었습니다.\r\n"))
}
func create(tableName string, whatTable int) string {
	// 맨끝에 봐라.. 이미 DB 열때부터 스미카 선택한것임
	//db, err := sql.Open("mysql", "root:1111@tcp(www.umanager.kr:3306)/map")
	db, err := sql.Open("mysql", "root:1111@tcp("+dbLocation+":"+db_port+")/map") // www.umanager.kr:3306
	if err != nil {
		panic(err)
	}
	defer db.Close()

	var fieldDefine string
	// 문법에 맞는 스트링만 넣어주면 동작한다. 스트링변수로 대체가능
	// 디버깅... 테이블 스트링 내용 받아보고 확인하도록 한다.
	switch whatTable {
	case 1:
		fieldDefine = apTableFieldDefine // ap 테이블
	case 2:
		fieldDefine = umTableFieldDefine // um 테이블
	case 3:
		fieldDefine = siteApTableFieldDefine // site 테이블
	case 4:
		fieldDefine = myidpw // PassWord 테이블
	default:
		fmt.Println("[ERROR] 해당하는 테이블 종류가 없습니다")
		return "[ERROR] 해당하는 테이블 종류가 없습니다"
	}
	// 테이블 존재 유무 조사 방법 : 테이블 없으면 이렇게 에러 뜬다 :: Error 1146: Table 'map.skn20161s224' doesn't exist

	/* disk & cpu 부하줄이기 운동
	var rowCnt int
	err = db.QueryRow("SELECT COUNT(*) FROM " + tableName + " WHERE id > ?", 0).Scan(&rowCnt)
	if err == nil {
		fmt.Println("[ERROR] 테이블 이름 중복 !! 현재 존재하는 동일 이름 테이블 row 수 : ", rowCnt, "\r\n")
		return "[ERROR] 동일이름 테이블 존재함"
	}*/

	// fmt.Println(err)
	// id 가 0보다 큰게 몇개 있나 물어봐서 있으면 테이블 존재도 확인하고 row 크기 정보도 알려주는 일석이조 !
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS " + tableName + fieldDefine)
	if err != nil {
		panic(err)
	}
	return ""
}
func site_makeTable(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var dat string
	if dateRange, ok := r.Form["yearmonth"]; !ok {
		fmt.Println("연월이 없습니다.. 사용법 확인하세요 - [젬피]", dateRange)
		fmt.Printf("%T  --------\n", dat)
		w.Write([]byte("연월이 없습니다.. 사용법 확인하세요 - [젬피]"))
		return
	}
	// 테이블 만든다
	var st string
	st = r.Form.Get("yearmonth")
	iNum, err := strconv.Atoi(st)
	// 형식 검사
	if err != nil {
		pp("쿼리 날짜 변환에 실패하였습니다. 테이블 생성하지 않습니다 -- [젬피]")
		w.Write([]byte("쿼리 날짜 변환에 실패하였습니다. 테이블 생성하지 않습니다 -- [젬피]"))
		return
	}
	// 자릿수 검사
	if len(st) != 6 {
		pp("연월 자릿수 오류 입니다. 테이블 생성하지 않습니다 -- [젬피]")
		w.Write([]byte("연월 자릿수 오류 입니다. 테이블 생성하지 않습니다 -- [젬피]"))
		return
	}
	// 범위 검사  :::  13월 14월등은 검사하지 않음 !!!!!!!!  td-1612
	if (iNum > 201812) || (iNum < 201612) {
		pp("연월 지정 범위를 초과하였습니다. 테이블 생성하지 않습니다 -- [젬피]")
		w.Write([]byte("연월 지정 범위를 초과하였습니다. 테이블 생성하지 않습니다 -- [젬피]"))
		return
	}
	st = "site" + st
	if create(st, 3) != "" {
		w.Write([]byte("[ERROR] 동일한 이름의 테이블이 있습니다. 테이블 만들지 않습니다.\r\n"))
		return
	}
	fmt.Printf("테이블이 정상적으로 만들어졌습니다. 만들어진 테이블 이름은 %s 입니다 !!\n", st)
	w.Write([]byte("테이블이 정상적으로 만들어졌습니다.\r\n"))
}
func event_makeTable(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()
	var dat string
	if dateRange, ok := r.Form["yearmonth"]; !ok {
		fmt.Println("연월이 없습니다.. 사용법 확인하세요 - [젬피]", dateRange)
		fmt.Printf("%T  --------\n", dat)
		w.Write([]byte("연월이 없습니다.. 사용법 확인하세요 - [젬피]"))
		return
	}
	// 테이블 만든다
	var st string
	st = r.Form.Get("yearmonth")
	iNum, err := strconv.Atoi(st)
	// 형식 검사
	if err != nil {
		pp("쿼리 날짜 변환에 실패하였습니다. 테이블 생성하지 않습니다 -- [젬피]")
		w.Write([]byte("쿼리 날짜 변환에 실패하였습니다. 테이블 생성하지 않습니다 -- [젬피]"))
		return
	}
	// 자릿수 검사
	if len(st) != 6 {
		pp("연월 자릿수 오류 입니다. 테이블 생성하지 않습니다 -- [젬피]")
		w.Write([]byte("연월 자릿수 오류 입니다. 테이블 생성하지 않습니다 -- [젬피]"))
		return
	}
	// 범위 검사  :::  13월 14월등은 검사하지 않음 !!!!!!!!  td-1612
	if (iNum > 201812) || (iNum < 201612) {
		pp("연월 지정 범위를 초과하였습니다. 테이블 생성하지 않습니다 -- [젬피]")
		w.Write([]byte("연월 지정 범위를 초과하였습니다. 테이블 생성하지 않습니다 -- [젬피]"))
		return
	}
	st = "um" + st
	if create(st, 2) != "" {
		w.Write([]byte("[ERROR] 동일한 이름의 테이블이 있습니다. 테이블 만들지 않습니다.\r\n"))
		return
	}
	fmt.Printf("테이블이 정상적으로 만들어졌습니다. 만들어진 테이블 이름은 %s 입니다 !!\n", st)
	w.Write([]byte("테이블이 정상적으로 만들어졌습니다.\r\n"))
}
func makeTable(w http.ResponseWriter, r *http.Request) {
	if gSystemLockStatus != "unlock" {
		w.Write([]byte("Your uManager System is LOCKED..."))
		print("Your uManager System is LOCKED...")
		time.Sleep(500)
		os.Exit(2965)
	}
	//fmt.Printf("Raw Data : ")
	//fmt.Println(r)
	//fmt.Println(r)
	// 파싱전 Form r.Form은 비어 있는 map이다. 파싱한다. 아그들 추출에 성공하면 비어 있는 map에 퀴리값들 map 안으로 들어가 있게된다.
	r.ParseForm()
	// 주소창 쿼리 내용만 뽑아서 프린트 한다.
	fmt.Printf("쿼리값들 : ")
	//fmt.Println(r.Form) // print form information in server side

	var dat string
	if dateRange, ok := r.Form["yearmonth"]; !ok {
		fmt.Println("연월이 없습니다.. 사용법 확인하세요 - [젬피]", dateRange)
		fmt.Printf("%T  --------\n", dat)
		w.Write([]byte("연월이 없습니다.. 사용법 확인하세요 - [젬피]"))
		return
	}
	/*
		if dateRange["yearmonth"] == nil {
			fmt.Println("연월이 없습니다.. 사용법 확인하세요 - [젬피]")
			w.Write([]byte("연월이 없습니다.. 사용법 확인하세요 - [젬피]"))
			return
		}
	*/
	// fmt.Println("생성 테이블 년월 : ", dateRange["yaermonth"])

	// 클라이언트 브라우저에 보여줄 HTML 헤더 설정. 이거 적용하면 일반 텍스트는 딱딱 붙어서 보인다.
	w.Header().Set("Content-Type", "text/html")
	//dat, err := ioutil.ReadFile("makeTable.html")
	//fmt.Println(dat)
	f, err := os.Open("makeTable.html")
	checkError(err)

	data := make([]byte, 10000) // 서버 하드에 있는 .html 파일 읽어들일 map 구조체
	count, err := f.Read(data)  // 이 한문장이 파일을 슬라이스 구조체에 옮기면서 글자수도 세고 에러도 체크한다
	if err != nil {
		log.Fatal(err)
	}
	var popUpMessage string
	//popUpMessage = fmt.Sprintf("read %d bytes: %q\n", count, data[:count])
	popUpMessage = fmt.Sprintf("%s", data[:count])
	w.Write([]byte(popUpMessage))

	/*
		fmt.Println("path", r.URL.Path)
		fmt.Println("scheme", r.URL.Scheme)
		fmt.Println(r.Form["url_long"])
	*/
	// 루프 돈 횟수와 파싱한 내용을 웹브라우져에 보내준다
	//w.Write([]byte(returnString))

	// 테이블 만든다
	var st string
	st = r.Form.Get("yearmonth")

	iNum, err := strconv.Atoi(st)
	fmt.Println(iNum)
	// 형식 검사
	if err != nil {
		pp("쿼리 날짜 변환에 실패하였습니다. 테이블 생성하지 않습니다 -- [젬피]")
		w.Write([]byte("쿼리 날짜 변환에 실패하였습니다. 테이블 생성하지 않습니다 -- [젬피]"))
		return
	}
	// 자릿수 검사
	if len(st) != 6 {
		pp("연월 자릿수 오류 입니다. 테이블 생성하지 않습니다 -- [젬피]")
		w.Write([]byte("연월 자릿수 오류 입니다. 테이블 생성하지 않습니다 -- [젬피]"))
		return
	}
	// 범위 검사  :::  13월 14월등은 검사하지 않음 !!!!!!!!  td-1612
	if (iNum > 201812) || (iNum < 201612) {
		pp("연월 지정 범위를 초과하였습니다. 테이블 생성하지 않습니다 -- [젬피]")
		w.Write([]byte("연월 지정 범위를 초과하였습니다. 테이블 생성하지 않습니다 -- [젬피]"))
		return
	}

	// DB 테이블 앞에 문자 반드시 필요
	st = "ap" + st

	if create(st, 1) != "" {
		w.Write([]byte("[ERROR] 동일한 이름의 테이블이 있습니다. 테이블 만들지 않습니다.\r\n"))
		return
	}
	fmt.Printf("테이블이 정상적으로 만들어졌습니다. 만들어진 테이블 이름은 %s 입니다 !!\n", st)
	w.Write([]byte("테이블이 정상적으로 만들어졌습니다.\r\n"))
}

type test_struct struct {
	Test string
}

func myPutHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Println("이거 HTML 아니다")
		//log.Fatal()
	}
	log.Println(r.Form)

	//LOG: map[{"test": "that"}:[]]
	var t test_struct
	for key, _ := range r.Form {
		log.Println(key)
		//LOG: {"test": "that"}
		err := json.Unmarshal([]byte(key), &t)
		if err != nil {
			log.Println(err.Error())
		}
	}
	log.Println(t.Test)
	//LOG: that

	//제이슨 디코더를 써서 body 내용 추출하기
	// decoder := json.NewDecoder(r.Body)
	// err := decoder.Decode(&ttt)
	//defer r.Body.Close()
	//log.Println(t.Test)

	fmt.Println(r.Form) // print form information in server side
	fmt.Println("path", r.URL.Path)
	fmt.Println("scheme", r.URL.Scheme)

	// 시리얼라이즈하는데 매우 우수한 방법임
	var myStr string
	myStr = fmt.Sprintf("%#v", r)
	w.Write([]byte("PUT 받음... Gorilla!\n"))
	w.Write([]byte(myStr))

	//	w.Write(r.Form)  // print form information in server side
	/*
		w.Write("path", r.URL.Path)
		w.Write("scheme", r.URL.Scheme)
	*/
}

// 부트플래그만 1이어도 일단 이쪽으로 보낸다
// site 테이블에 새로운 장비 추가할 목적
/*func setTrafic() {
	timenow := time.Now().String()
	traex, err := readLines("./log/" + timenow[0:7] + "-datatrafic.csv")
	//traex, err:=readLines("./log/2017-05-datatrafic.csv")
	if err != nil {
		fmt.Println(err)
	}
	if traex == nil {
		fmt.Println("비어있다.")
	} else {
		//fmt.Println(traex)
		for _, line := range traex {
			if line != "" && strings.Contains(line, "Mac") {
				//	fmt.Println(i, line)
				s := strings.Split(line, ",")
				//fmt.Println(s[0], "!!!")
				mac, txtra, rxtra := strings.Split(s[1], "="), strings.Split(s[2], "="), strings.Split(s[3], "=")
				txx, _ := strconv.ParseUint(txtra[1], 10, 64) // 10 진수로 int64 로 변환하기
				rxx, _ := strconv.ParseUint(rxtra[1], 10, 64) // 10 진수로 int64 로 변환하기
				//fmt.Println(mac[1], txx, rxx)
				rx[mac[1]] = rxx
				tx[mac[1]] = txx
				rxtxmap[mac[1]] = fmt.Sprint(s[0], ",Mac=", mac[1], ",tx=", tx[mac[1]], ",rx=", rx[mac[1]], "\n")
			}
		}
	}
	//tx[macAddr]=불라불라
}*/
var meshmacstring string
var wifimacto string
var txmap map[string]uint64
var hopcount map[string]int
var meshflagmap map[int]string
var rxtxmap map[string]string
var rxmap map[string]uint64
var tx map[string]uint64        //실제 보여지는거
var rx map[string]uint64        //실제 보여지는거
var syslogmac map[string]string //syslog find mac using ip
var datatrafic string
var fakemac = "00:06:7A:40:04:42"
var meshlist string
var meshfwupdateresult = 0
var meshconfw = "0"
var meshroot string
var meshmapstring map[string]string
var pingresult  map[string]bool
var errcheck	map[string]string

func talkingWithAp(w http.ResponseWriter, r *http.Request) {
	//
	var def string;
	//fmt.Println(r.RemoteAddr)
	var countt=0;
	if gSystemLockStatus != "unlock"&& pingpong!="OK" {
		print("AP 접근함.. 관리자 로그인이 필요합니다\n")
		return
	}
	/*****************	캐쉬 방지 헤더 설정	*******************/
	w.Header().Set("Cache-Control", "no-cache, private, max-age=0")
	w.Header().Set("Expires", time.Unix(0, 0).Format(http.TimeFormat))
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Accel-Expires", "0")
	r.ParseForm()
	var st string

	var ph string
	var queryError bool
	n := new(apTableVal)
	var returnString string // 브라우저에 리턴해줄 글 모음
	var returnString_Rec string
	var i int // 쿼리 추출 루프 도는 횟수 카운팅
	var magicFlag = false
	var modelName string


	var eth2rx,eth2tx uint64


	// map을 for 문에서 사용할때 항상 k, v 두가지 변수를 쓰고 이 변수는 map 변수를 따라가게된다. 덕타이핑~~
	for k, v := range r.Form {
		if k == "magic" { // 매직넘버 검사. 위치가 랜덤이므로 전수검사한다.
			if strings.Join(v, "") == "415$$" {
				magicFlag = true // 올바른 매직넘버 존재 확인 됨.. 이것은 DB에 저장할 필요가 없으므로 뺀다. 콘티뉴
				continue
			}

		}
		tempV := strings.Join(v, "") // 문자 map value 값을 string으로 변환하기
		switch k {
		case "bootFlag":
			t, _ := strconv.ParseInt(tempV, 10, 64) // 10 진수로 int8 로 변환하기
			n.bootFlag = uint8(t)
		case "macAddr":
			n.macAddr = tempV
		case "rxByteE3":
			t, _ := strconv.ParseInt(tempV, 10, 64) // 10 진수로 int64 로 변환하기
			eth2rx=uint64(t)
		case "macW1":
			n.macW1 = tempV
		case "macW0":
			n.macW0=tempV
		case "txByteE3":
			t, _ := strconv.ParseInt(tempV, 10, 64) // 10 진수로 int64 로 변환하기
			eth2tx=uint64(t)
		case "hostname":
			modelName = tempV
		case "modelName":
			modelName = tempV
		case "ipAddr":
			n.ipAddr = tempV
		case "meshidW1":
			n.meshId=tempV
		case "meshidW0":
			n.meshId0=tempV
		case "meshmodeW1":
			n.meshmode=tempV
		case "meshCnt":
			num1, _ := strconv.Atoi(tempV)
			n.nodecnt=num1
		case "signal":
			sig, _ := strconv.Atoi(tempV)
			n.signal=sig*-1
		case "btVer":
			n.btVer = tempV
		case "fwVer":
			n.fwVer = tempV
		case "ssidW0":
			n.ssidW0 = tempV
		case "ssidW1":
			n.ssidW1 = tempV
		case "chanW0":
			n.chanW0 = tempV
		case "chanW1":
			n.chanW1 = tempV
		case "rxByteW0":
			t, _ := strconv.ParseInt(tempV, 10, 64) // 10 진수로 int64 로 변환하기
			n.rxByteW0 = uint64(t)
		case "txByteW0":
			t, _ := strconv.ParseInt(tempV, 10, 64) // 10 진수로 int64 로 변환하기
			n.txByteW0 = uint64(t)
		case "rxByteW1":
			t, _ := strconv.ParseInt(tempV, 10, 64) // 10 진수로 int64 로 변환하기
			n.rxByteW1 = uint64(t)
		case "txByteW1":
			t, _ := strconv.ParseInt(tempV, 10, 64) // 10 진수로 int64 로 변환하기
			n.txByteW1 = uint64(t)                  // fmt.Println("txByteW1 is ", n.txByteW1)
		case "rxByteB0":
			t, _ := strconv.ParseInt(tempV, 10, 64) // 10 진수로 int64 로 변환하기
			n.rxByteB0 = uint64(t)
		case "txByteB0":
			t, _ := strconv.ParseInt(tempV, 10, 64) // 10 진수로 int64 로 변환하기
			n.txByteB0 = uint64(t)
		case "rxPktB0":
			//t, _ := strconv.ParseInt(tempV, 10, 64) // 10 진수로 int64 로 변환하기
			//rxPktB0 = uint64(t)
		case "txPktB0":
			//t, _ := strconv.ParseInt(tempV, 10, 64) // 10 진수로 int64 로 변환하기
			//txBPktB0 = uint64(t)
		case "rxPktW0":
			t, _ := strconv.ParseInt(tempV, 10, 64) // 10 진수로 int64 로 변환하기
			n.rxPktW0 = uint64(t)
		case "txPktW0":
			t, _ := strconv.ParseInt(tempV, 10, 64) // 10 진수로 int64 로 변환하기
			n.txPktW0 = uint64(t)
		case "rxPktW1":
			t, _ := strconv.ParseInt(tempV, 10, 64) // 10 진수로 int64 로 변환하기
			n.rxPktW1 = uint64(t)
		case "txPktW1":
			t, _ := strconv.ParseInt(tempV, 10, 64) // 10 진수로 int64 로 변환하기
			n.txPktW1 = uint64(t)
		case "assoCtW0":
			t, _ := strconv.ParseInt(tempV, 10, 64) // 10 진수로 int64 로 변환하기
			n.assoCtW0 = uint32(t)
		case "assoCtW1":
			t, _ := strconv.ParseInt(tempV, 10, 64) // 10 진수로 int64 로 변환하기
			n.assoCtW1 = uint32(t)
		case "devTemp":
			t, _ := strconv.ParseInt(tempV, 10, 64) // 10 진수로 int64 로 변환하기
			n.devTemp = int8(t)                     // 유일하게 signed 임 :::::  +, - 모두 필요함 ~~~ !!
		case "curMem":
			t, _ := strconv.ParseInt(tempV, 10, 64) // 10 진수로 int64 로 변환하기
			n.curMem = uint32(t)
		case "pingTime":
			t, _ := strconv.ParseInt(tempV, 10, 64) // 10 진수로 int64 로 변환하기
			n.pingTime = uint32(t)
		case "assoDevT":
			t, _ := strconv.ParseInt(tempV, 10, 64) // 10 진수로 int64 로 변환하기
			n.assoDevT = uint32(t)
		case "leaveDevT":
			t, _ := strconv.ParseInt(tempV, 10, 64) // 10 진수로 int64 로 변환하기
			n.leaveDevT = uint32(t)
		case "peerAp0":
			if tempV == "0" || tempV == "" {
				//nmac1 = "{\"source\": " + "\"" + zxcasd + "\"" + ", \"target\": " + "\"" + zxcasd + "\"}," + "\n"
				n.peerAp0 = "0"
			} else {
				/*abc:=strings.Replace(tempV,":","",-1)
				s1,_:=strconv.ParseUint(abc[8:10],16,32) //16진수를 10진수로 변환
				s2,_:=strconv.ParseUint(abc[10:12],16,32) //16진수를 10진수로 변환
				minus:=s2-2
				s3 := strconv.FormatUint(s1, 16)
				s4 := strconv.FormatUint(minus, 16)
				s3=strings.ToUpper(s3)
				s4=strings.ToUpper(s4)
				if len(s3)<2{
					s3="0"+s3
				}
				if len(s4)<2{
					s4="0"+s4
				}
				nmac1 = "{\"source\": " + "\"" + zxcasd + "\"" + ", \"target\": " + "\"" + tempV + "\"}," + "\n"
				*/
				n.peerAp0 = tempV
			}
		case "peerAp1":
			n.peerAp1 = tempV
			if tempV == "0" || tempV == "" {
				n.peerAp1 = "0"
			} else {
				/*abc:=strings.Replace(tempV,":","",-1)
				s1,_:=strconv.ParseUint(abc[8:10],16,32) //16진수를 10진수로 변환
				s2,_:=strconv.ParseUint(abc[10:12],16,32) //16진수를 10진수로 변환
				minus:=s2-3
				s3 := strconv.FormatUint(s1, 16)
				s4 := strconv.FormatUint(minus, 16)
				s3=strings.ToUpper(s3)
				s4=strings.ToUpper(s4)
				if len(s3)==1{
					s3="0"+s3
				}
				if len(s4)==1{
					s4="0"+s4
				}
				peer:=tempV[0:11]+":"+s3+":"+s4
				nmac2 = "{\"source\": " + "\"" + n.macAddr + "\"" + ", \"target\": " + "\"" + peer + "\"}," + "\n"
				*/
				n.peerAp1 = tempV
			}
			//////////////북마크
		/*case "staMac0":
			n.staMac0 = "00:00:00:00:00:00"
			if tempV == "0" || tempV == "" {
				//nmac3 = "{\"source\": " + "\"" + zxcasd + "\"" + ", \"target\": " + "\"" + zxcasd + "\"}," + "\n"
				n.staMac0 = "00:00:00:00:00:00"
			} else {
				//nmac3 = "{\"source\": " + "\"" + zxcasd + "\"" + ", \"target\": " + "\"" + tempV + "\",}," + "\n"
				// \"Value\": \"1\"
				n.staMac0 = "00:00:00:00:00:00"
			}*/
		default:
			//fmt.Println(k, tempV, "<-----------")
			if tempV!="0"{
				if strings.Contains(k,"staMac"){
					if strings.Contains(tempV,"/"){
						n.meshsigdef+=tempV+";"
						//fmt.Println(k," : ",v)
					}else{
						countt++
					}

				}else if strings.Contains(k,"Sig") {

				}else{
					def+= k+"="+tempV+","
				}
			}

			//queryError = true
		}

		i++ // 루프카운트
		// 쿼리를 DB에 반영하기 위함.  st 는 컬럼명 나열한것이고 ph는 값을 나열한 것이다.  mySQL 인서트를 위해 대응되는 스트링과 플레이스홀더에 넣을 값을 짝을 맞춰 만든다. csv 형식으로...
		st += k + ","
		ph += strings.Join(v, "") + "," // map 내에 있는 값은 []string 으로 정의되어 있어 string과는 호환되지 않음. 그리고 키값 하나에 하나의 []string 밖에 없으므로 조인해도 하나만 나온다. 여러게 있어야 Join 내부의 ""안 값이 의미를 가진다.. 결과적으로 ","을 뒤에 추가하여 csv 형식을 만든다.
		// fmt.Println(st,"<---->",ph) // 디버깅용

		// 클라이언트 브라우저에 보내기 위해 쿼리 내용 정리하여 스트링으로 모으는 작업
		if debug_level == "1" {
			inputQueryString += k + ": " + strings.Join(v, "") + "\n"
		}
	}

	if strings.Contains(table,n.macAddr)!=true{
		//fmt.Println("Unregistered",n.macAddr)
		//return
	}
	//fmt.Println("Full URL", r.Form)
	//fmt.Println(modelName)
	var peer1 string
	if len(n.peerAp0) > 5 {
		abc := strings.Replace(n.peerAp0, ":", "", -1)
		s1, _ := strconv.ParseUint(abc[8:10], 16, 32)  //16진수를 10진수로 변환
		s2, _ := strconv.ParseUint(abc[10:12], 16, 32) //16진수를 10진수로 변환
		minus := s2 - 2
		s3 := strconv.FormatUint(s1, 16)
		s4 := strconv.FormatUint(minus, 16)
		s3 = strings.ToUpper(s3)
		s4 = strings.ToUpper(s4)
		if len(s3) == 1 {
			s3 = "0" + s3
		}
		if len(s4) == 1 {
			s4 = "0" + s4
		}
		peer := n.peerAp0[0:11] + ":" + s3 + ":" + s4
		nmac1 = "{\"source\": " + "\"" + n.macAddr + "\"" + ", \"target\": " + "\"" + peer + "\"}," + "\n"
	} else {
		nmac1 = "{\"source\": " + "\"" + n.macAddr + "\"" + ", \"target\": " + "\"" + n.macAddr + "\"}," + "\n"
	}
	//5000시리즈는 peerA1의 mac값을 -2해야된다.
	if len(n.peerAp1) > 5 {
		var minus uint64
		abc := strings.Replace(n.peerAp1, ":", "", -1)
		s1, _ := strconv.ParseUint(abc[8:10], 16, 32)  //16진수를 10진수로 변환
		s2, _ := strconv.ParseUint(abc[10:12], 16, 32) //16진수를 10진수로 변환
		if strings.Contains(modelName,"MAP50")||strings.Contains(modelName,"MAP20"){
			minus = s2-2
		}else{
			minus=s2-3
		}

		s3 := strconv.FormatUint(s1, 16)
		s4 := strconv.FormatUint(minus, 16)
		s3 = strings.ToUpper(s3)
		s4 = strings.ToUpper(s4)
		if len(s3) == 1 {
			s3 = "0" + s3
		}
		if len(s4) == 1 {
			s4 = "0" + s4
		}
		peer := n.peerAp1[0:11] + ":" + s3 + ":" + s4
		//fmt.Println(n.macAddr,peer)
		peer1 = peer
		nmac2 = "{\"source\": " + "\"" + n.macAddr + "\"" + ", \"target\": " + "\"" + peer + "\"}," + "\n"
	} else {
		nmac2 = "{\"source\": " + "\"" + n.macAddr + "\"" + ", \"target\": " + "\"" + n.macAddr + "\"}," + "\n"
		peer1 = n.peerAp1
	}
	if n.staMac0 != "0" {
		nmac3 = "{\"source\": " + "\"" + n.macAddr + "\"" + ", \"target\": " + "\"" + n.staMac0 + "\"}," + "\n"
	} else {
		nmac3 = "{\"source\": " + "\"" + n.macAddr + "\"" + ", \"target\": " + "\"" + n.macAddr + "\"}," + "\n"
	}
	//fmt.Println(n.macAddr,peer)

	//fmt.Println(n.peerAp1,"/////////////",n.peerAp0,"////////////",n.ipAddr,"/",peer1)
	//fmt.Println(len(nmac1),nmac1,len(nmac2),nmac2,len(nmac3),nmac3)
	//var abc,def,ghi string
	//nodestringread:=strings.Replace(asdf,"{\"source\": \"\", \"target\": \"\"},","",1)
	//fmt.Println("{\"source\": \"\", \"target\": \"\"},\n!!!!!!!!!!!!!!!!!")
	//fmt.Println(len(nmac1),nmac1,len(nmac2),nmac2,len(nmac3),nmac3)
	///북마크7 노드 맥
	/*
	data, err := ioutil.ReadFile("nodeMac.txt")
	nodestringread := string(data)
	if err == nil {
		if strings.Contains(nodestringread, nmac1) != true && len(nmac1) > 63 { //nmac1은 맥과 노드 둘 다 포함한것이기 때문에 중복되면 알아서 안넣는다.
			nodestringread = nmac1 + nodestringread
			ioutil.WriteFile("nodeMac.txt", []byte(nodestringread), os.FileMode(644))
		} else if strings.Contains(nodestringread, nmac2) != true && len(nmac2) > 63 {
			nodestringread = nodestringread + nmac2
			ioutil.WriteFile("nodeMac.txt", []byte(nodestringread), os.FileMode(644))
		} else if strings.Contains(nodestringread, nmac3) != true && len(nmac3) > 63 {
			nodestringread = nodestringread + nmac3
			ioutil.WriteFile("nodeMac.txt", []byte(nodestringread), os.FileMode(644))
		}
	} else {
		err = ioutil.WriteFile("nodeMac.txt", []byte(""), os.FileMode(644))
		if err != nil {
			fmt.Println(err)
		}

	}*/
	// 직전 대비 트래픽 량 차이 계산
	mutex.Lock()
	errcheck[n.macAddr]=n.ipAddr
	mutex.Unlock()
	db, err := sql.Open("mysql", "root:1111@tcp("+dbLocation+":"+db_port+")/map")
	//rw, err:=db.Query("SHOW TABLE STATUS FROM map")
	//fmt.Println("11111111111111",rw,err,"aaaaasdasdoqwojeliahsdlh")
	if err != nil {
		panic(err)
	}

	defer db.Close()
	/************* 이전 ap 테이블에서 해당 MAC 값 소환하여 방금들어온 값과 차이 계산 **********/
	curYearMonth := time.Now().String()
	curYearBuf := curYearMonth[:4]                // 연도 짤라내기. 글자수는 0부터 시작하고 끝 숫자는 빠진다. 0,1,2,3 위치 잘라내기
	curYearMonth = curYearBuf + curYearMonth[5:7] // 월 짤라내기

//	ap := "ap" + curYearMonth
	//db.Exec("TRUNCATE TABLE "+ap) 테이블 삭제
	/*
		//////////////////이부분한번 없애보고 해보자. 렉걸리는지 확인 렉렉렉렉렉렉렉렉///////////////////////////////////////
		// ""사이트"" 테이블명 확정..
		fmt.Println("현재 선택된 테이블 : ",st) // 디버깅 중요정보

		b := new(apTableVal)  // get before data

		//d := new(apTableVal)  // get before data

		queryTrafficSt := "SELECT txByteW0, txByteW1, rxByteW0, rxByteW1 from " + ap + " WHERE macAddr = " + "\"" + n.macAddr + "\""
		queryTrafficSt += " order by time desc LIMIT 1"  // 300만개 row에서는 바로 랙걸린다... 못쓴다...

		err = db.QueryRow(queryTrafficSt).Scan(&b.txByteW0, &b.txByteW1, &b.rxByteW0, &b.rxByteW1)  // 무선 2개 모두 송수신 바이트 정보, 이전 데이타 꺼낸다
		if err != nil {
			if err == sql.ErrNoRows { // there were no rows, but otherwise no error occurred	// 처음 수신되서 ap 테이블에 mac 이 없을 수 있다...
				fmt.Println("플래그 확인중... 해당하는 MAC 주소가 없습니다 (신규수신)", sql.ErrNoRows)
				w.Write([]byte("@@@ DB 쿼리 에러 -- 해당하는 MAC 주소가 없습니다\n"))
			} else {
				fmt.Println("직전 DB 값 읽어오지 못했습니다", sql.ErrNoRows)
				//log.Fatal(err)
			}
		}
	*///queryTrafficSt := "SELECT txByte, rxByte from " + ap + " WHERE macAddr = " + "\"" + n.macAddr + "\""
	//fmt.Println(queryTrafficSt)
	//fmt.Println("===> ", n.txByteW0, n.txByteW1, n.rxByteW0, n.rxByteW1)  // 들어온 트래픽 내용
	///////////////////////////////////////////////////////////////////////////////////////////////////////**********************************북마크북마크북마크************///////////////////////////////////
	/*
		println(b.txByteW0, b.txByteW1, b.rxByteW0, b.rxByteW1)  // DB 값 확인
		println(n.txByteW0, n.txByteW1, n.rxByteW0, n.rxByteW1)  // 이번에 들어온 값 확인
		d.txByteW0 = n.txByteW0 - b.txByteW0
		d.txByteW1 = n.txByteW1 - b.txByteW1
		d.rxByteW0 = n.rxByteW0 - b.rxByteW0
		d.rxByteW1 = n.rxByteW1 - b.rxByteW1
		println(d.txByteW0, d.txByteW1, d.rxByteW0, d.rxByteW1)  // 둘사이 차이 확인

		topCounters.downloadTraffic += d.txByteW0 + d.txByteW1
		topCounters.uploadTraffic += d.rxByteW0 + d.rxByteW1
	*/
	/*
	a00_value := topCounters.downloadTraffic >> 20 // 1024 * 1024 Byte로 나눔 (MB 단위)
	a01_value := topCounters.uploadTraffic >> 20
	// 웹소켓으로 보내기
	returnString = fmt.Sprint("-A00-", fmt.Sprintf("%08d", a00_value))
	returnString += fmt.Sprint("-A01-", fmt.Sprintf("%08d", a01_value))
	*/
	//fmt.Println(n.staMac0,"/////////////")
	time.Sleep(time.Millisecond * 50)
	mac := strings.Replace(n.macAddr, ":", "", -1)
	macTag := mac[len(mac)-6:]
	if strings.Contains(modelName,"MAP9200"){
		n.rxByteB0+=eth2rx
		n.txByteB0+=eth2tx
	}
	//sig, _ := strconv.Atoi("-50")
	//n.signal=sig*-1

	//n.meshId="TEST" //북마크 메쉬아이디 들어오면 맥태그 말고 메쉬아이디 넣자
	if strings.Contains(meshlist,n.meshId)!=true{
		meshlist+=n.meshId+","
	}
	//fmt.Println(meshlist)
	if len(n.meshId)>3{
		if strings.Contains(wifimacto,n.macAddr){

		}else{
			wifimacto+="/"+n.macW1+"="+n.macAddr
		}
	//fmt.Println(n.macAddr,"////",n.macW1,"////",n.meshId,"////",n.meshmode,"////",n.nodecnt,"////",def)
		meshinfo(n.macW1,def,n.nodecnt)
		if n.meshmode=="4"{
			if strings.Contains(meshroot,n.macAddr){
				meshroot+=n.macAddr+","
			}

			hopcount[n.macAddr]=0;
			createtopology(n.macW1,n.nodecnt,def)
			parsfunction(n.meshId,n.ipAddr,n.macAddr,n.macW1,"1000")
		}else{
			signal := findnextwifimac(n.macW1,n.meshsigdef)
			parsfunction(n.meshId,n.ipAddr,n.macAddr,n.macW1,signal)
		}
		//fmt.Println(n.signal,"신호세기")

	}
	if strings.Contains(modelName,"327680006"){
		modelName="5000_MESH"
	}
	//if strings.Contains() 9200 메쉬 추가


	realtx,realrx,totaltx,totalrx:=readTr(macTag,n.rxByteB0,n.txByteB0)
	writeTr(n.macAddr,totaltx,totalrx)

	//var txByte,rxByte uint64
	//var totaltx,totalrx uint64
	//fmt.Println(n.macAddr,"==",n.txByteB0,"//////",n.rxByteB0)
	//totaltx=tx[n.macAddr]
	//totalrx=rx[n.macAddr]
	/*if len(n.macAddr) != 0 {
		txpast := txmap[n.macAddr]
		rxpast := rxmap[n.macAddr]
		//fmt.Print(n.macAddr, ",",txpast)
		//fmt.Println(",",rxpast)
		//txByte:=n.txByteW0+n.txByteW1
		//rxByte:=n.rxByteW0+n.rxByteW1
		if modelName=="MAP5010"||modelName=="MAP2010"||modelName=="MAP9200"{
			txByte = n.txByteB0
			rxByte = n.rxByteB0
			//fmt.Println(n.txByteB0,"/////",n.rxByteB0)
		//}else{ //아크랩스 때문에 넣은거
		//	txByte = n.txByteB0 + n.txByteW0 + n.txByteW1
		//	rxByte = n.rxByteB0 + n.rxByteW0 + n.rxByteW1
		}
		if txpast != 0 || rxpast != 0 {
			if txpast < txByte {
				totaltx += (txByte - txpast) //지금 - 예전값

			} else if txpast > txByte {
				totaltx += txByte
				//realtx = txByte
			}
			///////////////tx
			if rxpast < rxByte {
				totalrx+= (rxByte - rxpast)
				//realrx = rxByte - rxpast
			} else if rxpast > rxByte {
				totalrx += rxByte
				//realrx = rxByte
			}
		}
		tx[n.macAddr]=totaltx
		rx[n.macAddr]=totalrx
		timenow := time.Now().String()
		//fmt.Println(txpast,rxpast)//AP가 주는 트래픽
		//fmt.Println("===> ", txByte, rxByte) // 들어온 트래픽 내용
		//fmt.Println(realtx,realrx)//실제 트래픽
		//fmt.Println(txByte - txpast)
		txmap[n.macAddr] = txByte
		rxmap[n.macAddr] = rxByte
		rxtxmap[n.macAddr] = fmt.Sprint(timenow[0:19]+",Mac=", n.macAddr, ",tx=", tx[n.macAddr], ",rx=", rx[n.macAddr], "\n")
		vall,err:=readLines("./log/"+timenow[0:7]+"-datatrafic.csv")
		var putstring string
		if err!=nil{
			fmt.Println("err")
		}
		i:=0
		cont:=false
		for _,line:=range vall{
			if strings.Contains(line,n.macAddr){
				putstring+=rxtxmap[n.macAddr]
				cont=true
			}else{
				putstring+=vall[i]+"\n"
			}
			i++
		}
		if cont==false{
			putstring+=n.macAddr
		}
		ioutil.WriteFile("./log/"+timenow[0:7]+"-datatrafic.csv",[]byte(putstring),os.FileMode(644))
		/*if strings.Contains(vall, timenow[0:7]) {
			err = ioutil.WriteFile("./log/"+timenow[0:7]+"-datatrafic.csv", []byte(vall), os.FileMode(644))
		}
		if err != nil {
			fmt.Println(err)
		}
		vall = ""*/
	/*} else {
		fmt.Println("잘못된 Mac이 입력되었습니다.")
		return
	}*/

	//fmt.Println(rxtxmap)

	//fmt.Print(n.macAddr, ",",txmap[n.macAddr])
	//fmt.Println(",",rxmap[n.macAddr])
	//fmt.Println(txmap,rxmap,"맵안에 저장된 변수")
	/*
		//fmt.Println(n.rxByteW0,n.txByteW0,n.rxByteW1,n.txByteW1,"//////////",n.rxByteB0,n.txByteB0,n.rxPktB0,n.txPktB0)//북마크
		var rr,tt uint64
		totaltrafic:=fmt.Sprint(n.macAddr,",rr=",rr,",tt=",tt,",rxByte=",rxByte,",txByte=",txByte,"\n")


		timenow := time.Now().String()
		traex, err:=ioutil.ReadFile("./log/"+timenow[0:7]+"-datatrafic.csv")
		if err != nil {
			fmt.Println(err)
		}
		traex1:=string(traex)
		if strings.Contains(traex1,n.macAddr)!=true {
			fmt.Println(traex1)
			traex1 += totaltrafic
			err = ioutil.WriteFile("./log/"+timenow[0:7]+"-datatrafic.csv", []byte(traex1), os.FileMode(644))
			if err != nil {
				fmt.Println(err)
			}
		}else if strings.Contains(traex1,n.macAddr)==true{

		}

		input, err := ioutil.ReadFile("./log/"+timenow[0:7]+"-datatrafic.csv")
		if err != nil {
			log.Fatalln(err)
		}
		lines := strings.Split(string(input), "\n")

		for i, line := range lines {
			if strings.Contains(line, n.macAddr) {
				strings.Split(lines[i],",")
			}
		}
		output := strings.Join(lines, "\n")
		err = ioutil.WriteFile("myfile", []byte(output), 0644)
		if err != nil {
			log.Fatalln(err)
		}

	*/
	//timenow := time.Now().String()
	/*
		totaltrafic:=fmt.Sprint(n.macAddr,"-----","tt=",tt,",rr=",rr,",ttt=",ttt,",rrr=",rrr,",tx=",txByte,",rx=",rxByte,"-----",n.macAddr,"----->","\n")

		//fmt.Println(totaltrafic)
		traex, err:=ioutil.ReadFile("./log/"+timenow[0:7]+"-datatrafic.csv")
		if err != nil {
			fmt.Println(err)
		}
		traex1:=string(traex)
		traex1+=totaltrafic
		//fmt.Println(traex1)
		a11:=strings.Index(string(traex),n.macAddr)
		a22:=strings.LastIndex(string(traex),n.macAddr)
		//fmt.Println(a11,a22)

	//todo AP 수 늘어날 때 문제가 생기는지? 글로벌 변수로 만들어서 저장 시간도 조정.
		if strings.Contains(string(traex),n.macAddr)!=true{
			fmt.Println(traex1)
			err=ioutil.WriteFile("./log/"+timenow[0:7]+"-datatrafic.csv", []byte(traex1), os.FileMode(644))
			if err != nil {
				fmt.Println(err)
			}
		}else if strings.Contains(string(traex),n.macAddr)==true{
			rxtx:=traex1[a11+22:a22-5]
			//fmt.Println(rxtx)
			rt:=strings.Split(rxtx,",")//rt[0]=tt=0 rt[1]=rr=0 rt[2]=ttt=0 rt[3]=rrr=0 rt[4]=tx=0 rt[5]=rx=0
			tt:=strings.Split(rt[0],"=")
			rr:=strings.Split(rt[1],"=")
			ttt:=strings.Split(rt[2],"=")
			rrr:=strings.Split(rt[3],"=")
			//tx:=strings.Split(rt[4],"=")
			//rx:=strings.Split(rt[5],"=")
			//fmt.Println(tt[1],rr[1],"예전값")
			//fmt.Println(txByte,rxByte,"지금 값")
			tt1, err := strconv.ParseUint(tt[1], 10, 64)//저장된 값 즉 예전값
			rr1, err := strconv.ParseUint(rr[1], 10, 64)
			ttt1, err = strconv.ParseUint(ttt[1], 10, 64)//저장된 값이 더 클 때 저장장소
			rrr1, err = strconv.ParseUint(rrr[1], 10, 64)
			//tx1, err := strconv.ParseUint(tx[1], 10, 64)//더해진거?
			//rx1,err:=strconv.ParseUint(rx[1], 10, 64)//예전값

			//fmt.Println(tt1,rr1,ttt1,rrr1,tx1,rx1)

			if tt1>txByte{
				ttt1=ttt1+tt1
				fmt.Println("tx초기화됬어요",ttt1)
			}
			if rr1>rxByte{
				rrr1=rrr1+rr1
				fmt.Println("rx초기화됬어용",rrr1)
			}
			//fmt.Println(tx1+txByte,rx1+rxByte)
			realtrafic:=fmt.Sprint("tt=",txByte,",rr=",rxByte,",ttt=",ttt1,",rrr=",rrr1,",tx=",txByte+ttt1,",rx=",rxByte+rrr1)
			//fmt.Println(realtrafic)
			traex1=strings.Replace(string(traex),rxtx,realtrafic,-1)
			err=ioutil.WriteFile("./log/"+timenow[0:7]+"-datatrafic.csv", []byte(traex1), os.FileMode(644))
			if err != nil {
				fmt.Println(err)
			}

		}
-
	*/
	if debug_level=="websoc"{
		fmt.Println(countwebsoc)
	}
	//fmt.Println(modelName)
		//fmt.Println(n.macAddr, "는 ", peer1, "에 붙었다. \n실시간 tx값은 ", realtx, "이고\n 실시간 rx값은 ", realrx, "이다.")
		if countwebsoc<20{
			if n.bootFlag==4{

			}else{
				countstring:=strconv.Itoa(countt)
				//fmt.Println()
				returnString_Rec = fmt.Sprint("부:", n.bootFlag, ",", "맥:", n.macAddr, ",", "아:", n.ipAddr, ",", "컴:", spend_time, ",", "상태:", fmt.Sprint(time.Now())[:19], ",", "SSID1: ", n.ssidW0, ",", "SSID2:", n.ssidW1, ", 채널: ", n.chanW1, ", 전송: ", totaltx, ", 수신: ", totalrx, ", 실전: ", realtx, ", 실수: ", realrx, ",  Stalist:", countstring, ", 모델: ", modelName, ", 온도: ", n.devTemp, "; ") //불라불라;시간
				//소켓에 보내는 스트링 안에는 ,로 구분하고 스트링 사이는 ;로 구분하자
				webSocString_Rec += returnString_Rec + "\n"
				webSocString_Rec_time = fmt.Sprint(webSocString_Rec, spend_time,"[]",meshlist) //불라불라;시간
				webSocString_map += fmt.Sprint("부:", n.bootFlag, ",", "맥:", n.macAddr, ",", "아:", n.ipAddr, ",", "컴:", spend_time, ",", "상태:", fmt.Sprint(time.Now())[:19], ",", "SSID1: ", n.ssidW0, ",", "SSID2:", n.ssidW1, ", 채널: ", n.chanW1, ", 전송: ",totaltx, ", 수신: ", totalrx, ", 실전: ", realtx, ", 실수: ", realrx, ", 붙맥: ", peer1, ", 모델: ", modelName, ", 온도: ", n.devTemp, "; ", spend_time) //불라불라;시간
				//*************여기서 데이터 넣는다. **************************
				//returnString = fmt.Sprint("-A05-", fmt.Sprintf("%03d", gApCnt), "[", n.macAddr+"]-", n.ipAddr, " => ", (fmt.Sprint(time.Now()))[:19], ", 부트: ", n.bootFlag, ", Ver: ", n.fwVer, ", SSID: ", n.ssidW1, ", 채널: ", n.chanW1, ", 전송: ", n.txByteW1, ", 수신: ", n.rxByteW1, ", 온도: ", n.devTemp, "...") // 유저화면 한줄 만들기..한땀한땀..
				returnString = fmt.Sprint("[", n.macAddr+"]-", n.ipAddr, " => ", (fmt.Sprint(time.Now()))[:19], ", 부트: ", n.bootFlag, ", Ver: ", n.fwVer, ", SSID: ", n.ssidW0, ", 채널: ", n.chanW1, ", 전송: ", n.txByteW1, ", 수신: ", n.rxByteW1, ", 온도: ", n.devTemp, "...") // 유저화면 한줄 만들기..한땀한땀..
				webSocString += returnString + "\n"
			}
		}else{

		}
	countwebsoc++                                                                                                                                                                                                        // AP 많으면 여려개 모아야 한다.  웹소켓 주기 보다 빠를테니까

	/********************   url 값 파싱 완료 !!!  *****************************/
	/*//err = ioutil.WriteFile("./file_storage/Apdata.txt", []byte(webSocString), os.FileMode(644))
	if err != nil {
		panic(err)
	}
	datt, _:=ioutil.ReadFile("Apdata.txt")
	dbstring:=string(datt)
	webSocString+=dbstring
	*/
	if n.macAddr == "" { // MAC 없이 엉뚱한 쿼리 들어온경우 걍 리턴..
		fmt.Println("\n쿼리에 MAC 주소가 없습니다.. 아무것도 안합니다.\n")
		w.Write([]byte("\n쿼리에 MAC 주소가 없습니다.. 아무것도 안합니다.\n"))
		return
	}
	//fmt.Println(meshmacstring)
	check2:=checkmeshflag2(meshmacstring)
	switch n.bootFlag { //  부팅플래그는 단순 이벤트 표출, 기록할 뿐.. ::: 여기서 event 테이블을 조작할 필요하 있다.. 주요 이벤트를 기록해 놓기 똵 좋은 위치..
	case 1:
		fmt.Println("\n*******************>>> AP가 조금전 부팅하였습니다... MAC:", n.macAddr, " -- IP:", n.ipAddr)
	//	webSocString = webSocString + "\n>>> AP가 조금전 부팅하였습니다... MAC: " + n.macAddr + "IP: " + n.ipAddr + "\n" // 부팅한 AP 갯수 만큼 모은다.. 실제로은 하나이상 되기 어렵지만..
	case 2:
		fmt.Println("\n*******************>>> AP가 다운로드 시작합니다... MAC:", n.macAddr, " -- IP:", n.ipAddr)
	case 3:
		fmt.Println(modelName,"\n*******************>>> (다운로드 완료) AP가 펌웨어 다운로드를 완료했습니다... MAC:", n.macAddr, " -- IP:", n.ipAddr)
		//	webSocString = webSocString + "\n>>> AP가 조금전 부팅하였습니다... MAC: " + n.macAddr + "IP: " + n.ipAddr + "\n" // 부팅한 AP 갯수 만큼 모은다.. 실제로은 하나이상 되기 어렵지만..
		if check2==0{
			fmt.Println("장비 펌웨어 쓰기 준비 완료")
			//sethopmeshflag(maxhop,meshㅣconfw)
		}
	case 4:
		fmt.Println(modelName,"\n*******************>>> (다운로드 완료) AP가 쓰기 대기중입니다... MAC:", n.macAddr, " -- IP:", n.ipAddr)
		return
	case 5:
		fmt.Println("\n*******************>>> (전원유지 !!) AP가 펌웨어를 업데이트 합니다... MAC:", n.macAddr, " -- IP:", n.ipAddr)
		if strings.Contains(modelName,"MESH"){
			//fmt.Println("\n*******************>>> (다운로드 시작) AP가 펌웨어 업데이트를 시작합니다... 본 장비는 MESH 장비입니다.")
			meshfwupdateresult--
			if(meshfwupdateresult==0&&maxhop!=0){
				maxhop--
				if meshflagmap[maxhop]==""{
					maxhop--
				}
				sethopmeshflag(maxhop,meshconfw)
			}else if(meshfwupdateresult==0&&maxhop==0){
				fmt.Println("플래그가 모두 0이되었습니다.")
				meshmacstring=""
				meshconfw="0"
			}
		}

	//	webSocString = webSocString + "\n>>> AP가 조금전 부팅하였습니다... MAC: " + n.macAddr + "IP: " + n.ipAddr + "\n" // 부팅한 AP 갯수 만큼 모은다.. 실제로은 하나이상 되기 어렵지만..
	//	webSocString = webSocString + "\n>>> AP가 조금전 부팅하였습니다... MAC: " + n.macAddr + "IP: " + n.ipAddr + "\n" // 부팅한 AP 갯수 만큼 모은다.. 실제로은 하나이상 되기 어렵지만..
	}

//	if len(n.ipAddr) > 8 {
//		mac := strings.Replace(n.macAddr, ":", "", -1)
//		macTag := mac[len(mac)-6:]
//		if syslogmac[n.ipAddr] != macTag {
			//fmt.Println(syslogmac[n.ipAddr], "----->", macTag, "로 변경")
//			syslogmac[n.ipAddr] = macTag
			//fmt.Println(syslogmac)

//		} else {
			//fmt.Println(syslogmac[n.ipAddr],"----->",macTag)
			//
//		}
//	}
	//syslog find mac using ip
	/************* site 테이블에서 기존 MAC 찾아보고 그 결과에 따라 자동등록등 결정.. 우선 현재 시각에 해당하는 테이블 월을 확정한다 **********/
	curYearMonth = time.Now().String()
	curYearBuf = curYearMonth[:4]                 // 연도 짤라내기. 글자수는 0부터 시작하고 끝 숫자는 빠진다. 0,1,2,3 위치 잘라내기
	curYearMonth = curYearBuf + curYearMonth[5:7] // 월 짤라내기
	st = "site" + curYearMonth                    // ""사이트"" 테이블명 확정.. fmt.Println("현재 선택된 테이블 : ",st) // 디버깅 중요정보
	/****************** 신규발견시 자동 등록 ********************/
	db, err = sql.Open("mysql", "root:1111@tcp("+dbLocation+":"+db_port+")/map")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	if n.chanW0=="0"{
		n.chanW0="-"
	}

	s := new(siteApTableVal)
	querySt := "SELECT macAddr from " + st + " WHERE macAddr = " + "\"" + n.macAddr + "\"" /// 테이블이 전혀 다르다!!! 하지만 물어볼땐 기존 n 구조체 MAC을 사용
	//fmt.Println("===", querySt)   // 앞으로 모두 이렇게 코딩하자... 그래야 디버깅 된다. "보이게하라. 문제지역을 조명탄으로.."
	err = db.QueryRow(querySt).Scan(&s.macAddr)

	//fmt.Println("================", s.macAddr)   // null 일경우 DB에 없는 신규 MAC 이다..
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("새로운 MAC 발견하여 신규 장비등록 진행합니다...", sql.ErrNoRows)
			webSocString = webSocString + ">>>>>> (!) 새로운 MAC 발견하여 신규 장비등록 진행합니다... MAC = " + n.macAddr + "\n" // 웹소켓으로 전달 준비

			// 이제 진짜로 site DB에 넣는다....
			stmtIns, err := db.Prepare("INSERT INTO " + st + " " + siteTableInsert) // 테이블명 + 집어넣는 필드와 플레이스 홀더. INTO 뒤에 그리고 st 다음에도 공백하나 넣어라..
			if err != nil {
				fmt.Println("사이트 테이블 인써트 준비 하는데서 에러발생!!!!!--[시스템 점검 필요 !!]")
			}
			defer stmtIns.Close()

			s.id = n.macAddr

			s.macAddr = n.macAddr
			s.location = n.fwVer
			s.modelName = modelName
			s.time = time.Now().String()[0:16] // 현재시각 받기
			s.lastRx = uint32(time.Now().Unix())
			s.ipAddr = n.ipAddr
			s.flagFirm = "0" // 등록단계에서 DB에 1을 기록 할 수 없고...
			s.flagConf = "0" // DB에 1을 기록 할 수 없고...
			if n.chanW0=="0"{
						n.chanW0="-"
			}
			s.urlFirm = n.chanW0
			s.urlConf = n.chanW1

			/*
			if strings.Contains(modelName,"MAP20"){

				s.urlFirm=n.chanW1
				s.urlConf=n.chanW0
			}else{

			}*/

			s.devDesc = n.meshId

			// n.id 대신 n.macAddr 로 변경한다.
			_, err = stmtIns.Exec(s.id, s.time, s.macAddr, s.location, s.modelName, s.lastRx, s.ipAddr, s.flagFirm, s.flagConf, s.urlFirm, s.urlConf, s.devDesc)
			checkError(err)

			// DB에서만 잘 관리하고 있으면 자동 정렬된 화면 출력기능으로 첵빡 테이블 파일 만들수 있으니까 당분간 ok

			// ROW 갯수 확인하는 방법

			// disk & cpu 부하줄이기 운동
			webSocString += "-A05-" + fmt.Sprintf("%03d", gApCnt)
			newmonthap = 0
		} else {
			newmonthap++
			fmt.Println("AP 관리패킷 수신했으나 DB에서 해당 MAC 확인작업은 실패했습니다 (site DB 시스템 점검 필요)", sql.ErrNoRows)
			if newmonthap == 3 {
				go open("http://" + db_url + ":8080/site_maketable?yearmonth=" + curYearMonth)
				os.RemoveAll("./tmp")
				//osfile,_ = os.OpenFile("asdf.txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
			}
		}
	} else { // 매번 apTalking 쿼리 받을때 마다 실행하여 검사 한다.... 왜? MAC은 DB에 있으나 IP가 변경된 경우 업데이트 하기 위함
		newmonthap = 0
		//querySt = "SELECT flagFirm, flagConf FROM " + st + " WHERE macAddr = " + "\"" + n.macAddr + "\""
		ipQuerySt := "SELECT ipAddr, modelName,location,flagFirm,flagConf,urlFirm,urlConf,devDesc from " + st + " WHERE macAddr = " + "\"" + n.macAddr + "\"" /// 물어볼땐 기존 n 구조체 MAC을 사용 북마크북마크
		if debug_level == "1" {
			fmt.Println("===>>", ipQuerySt) // 앞으로 모두 이렇게 코딩하자... 그래야 디버깅 된다. "보이게하라. 문제지역을 조명탄으로.."
		}
		err = db.QueryRow(ipQuerySt).Scan(&s.ipAddr, &s.modelName, &s.location, &s.flagFirm, &s.flagConf,&s.urlFirm,&s.urlConf,&s.devDesc) // s 구조체변수로 받아낸다
		if err != nil { // 아예 ip 자체가 스캔안될 경우
			print("DB에 MAC은 등록되어 있으나 IP 혹은 ModelName은 찾지 못하는 버그발생...")
		} else {
			if n.ipAddr != s.ipAddr { // 스캔한 ip와 들어온 ip 비교해서 같은지 판단
			updateQuerySt := "update " + st + " set ipAddr = '" + n.ipAddr + "' WHERE macAddr = '" + n.macAddr + "'" // ip를 AP에서 들어온 값으로 업데이트 한다
			// UPDATE Orders SET FirstName = 'John', LastName = 'Who' WHERE LastName='Wo'
			fmt.Println(n.macAddr, "AP의 IP가 변경되었습니다.. DB 값 수정합니다..", updateQuerySt)

			up, err := db.Prepare(updateQuerySt)
			if err != nil {
				panic(err)
			}
			defer up.Close()
			_, err = up.Exec()
			if err != nil {
				// the query, Scan returns ErrNoRows. 조건에 맞는 row가 없으면 프로그램이 종료되버림. ㅠ.ㅠ
				if err == sql.ErrNoRows {
					// there were no rows, but otherwise no error occurred
					fmt.Println("DB에 IP 주소 쓰다가 에러 발생 -- [젬피]", sql.ErrNoRows)
				} else {
					fmt.Println("해당 IP 주소의 row가 없습니다 !! -- [젬피]", sql.ErrNoRows)
					//log.Fatal(err)
				}
			}
		}
		if modelName != s.modelName { // 스캔한 MN과 들어온 MN 비교해서 같은지 판단
			if modelName == "unKnown" {
				modelName = s.modelName
			} else {
				updateQuerySt := "update " + st + " set modelName = '" + modelName + "' WHERE macAddr = '" + n.macAddr + "'" // ip를 AP에서 들어온 값으로 업데이트 한다
				// UPDATE Orders SET FirstName = 'John', LastName = 'Who' WHERE LastName='Wo'
				fmt.Println(n.macAddr, "AP의 modelName이 변경되었습니다.. DB 값 수정합니다..", updateQuerySt)
				up, err := db.Prepare(updateQuerySt)
				if err != nil {
					panic(err)
				}
				defer up.Close()
				_, err = up.Exec()
				if err != nil {
					// the query, Scan returns ErrNoRows. 조건에 맞는 row가 없으면 프로그램이 종료되버림. ㅠ.ㅠ
					if err == sql.ErrNoRows {
						// there were no rows, but otherwise no error occurred
						fmt.Println("DB에 modelName 쓰다가 에러 발생 -- [젬피]", sql.ErrNoRows)
					} else {
						fmt.Println("해당 modelName의 row가 없습니다 !! -- [젬피]", sql.ErrNoRows)
						//log.Fatal(err)
					}
				}
			}
		}
		if n.chanW0 != s.urlFirm {
				updateQuerySt := "update " + st + " set urlFirm = '" + n.chanW0 + "' WHERE macAddr = '" + n.macAddr + "'" // ip를 AP에서 들어온 값으로 업데이트 한다
				// UPDATE Orders SET FirstName = 'John', LastName = 'Who' WHERE LastName='Wo'
				fmt.Println(n.macAddr, "AP의 chanW0 이 변경되었습니다.. DB 값 수정합니다..", updateQuerySt)
				up, err := db.Prepare(updateQuerySt)
				if err != nil {
					panic(err)
				}
				defer up.Close()
				_, err = up.Exec()
				if err != nil {
					// the query, Scan returns ErrNoRows. 조건에 맞는 row가 없으면 프로그램이 종료되버림. ㅠ.ㅠ
					if err == sql.ErrNoRows {
						// there were no rows, but otherwise no error occurred
						fmt.Println("DB에 FW 쓰다가 에러 발생 -- [젬피]", sql.ErrNoRows)
					} else {
						fmt.Println("해당 FW의 row가 없습니다 !! -- [젬피]", sql.ErrNoRows)
						//log.Fatal(err)
					}
				}
			}
			if n.chanW1 != s.urlConf {
				updateQuerySt := "update " + st + " set urlConf = '" + n.chanW1 + "' WHERE macAddr = '" + n.macAddr + "'" // ip를 AP에서 들어온 값으로 업데이트 한다
				// UPDATE Orders SET FirstName = 'John', LastName = 'Who' WHERE LastName='Wo'
				fmt.Println(n.macAddr, "AP의 chanW1 이 변경되었습니다.. DB 값 수정합니다..", updateQuerySt)
				up, err := db.Prepare(updateQuerySt)
				if err != nil {
					panic(err)
				}
				defer up.Close()
				_, err = up.Exec()
				if err != nil {
					// the query, Scan returns ErrNoRows. 조건에 맞는 row가 없으면 프로그램이 종료되버림. ㅠ.ㅠ
					if err == sql.ErrNoRows {
						// there were no rows, but otherwise no error occurred
						fmt.Println("DB에 FW 쓰다가 에러 발생 -- [젬피]", sql.ErrNoRows)
					} else {
						fmt.Println("해당 FW의 row가 없습니다 !! -- [젬피]", sql.ErrNoRows)
						//log.Fatal(err)
					}
				}
			}
		if n.fwVer != s.location {
			updateQuerySt := "update " + st + " set location = '" + n.fwVer + "' WHERE macAddr = '" + n.macAddr + "'" // ip를 AP에서 들어온 값으로 업데이트 한다
			// UPDATE Orders SET FirstName = 'John', LastName = 'Who' WHERE LastName='Wo'
			fmt.Println(n.macAddr, "AP의 fwVer이 변경되었습니다.. DB 값 수정합니다..", updateQuerySt)
			up, err := db.Prepare(updateQuerySt)
			if err != nil {
				panic(err)
			}
			defer up.Close()
			_, err = up.Exec()
			if err != nil {
				// the query, Scan returns ErrNoRows. 조건에 맞는 row가 없으면 프로그램이 종료되버림. ㅠ.ㅠ
				if err == sql.ErrNoRows {
					// there were no rows, but otherwise no error occurred
					fmt.Println("DB에 FW 쓰다가 에러 발생 -- [젬피]", sql.ErrNoRows)
				} else {
					fmt.Println("해당 FW의 row가 없습니다 !! -- [젬피]", sql.ErrNoRows)
					//log.Fatal(err)
				}
			}
		}
		if n.meshId != s.devDesc {
				updateQuerySt := "update " + st + " set devDesc = '" + n.meshId + "' WHERE macAddr = '" + n.macAddr + "'" // ip를 AP에서 들어온 값으로 업데이트 한다
				// UPDATE Orders SET FirstName = 'John', LastName = 'Who' WHERE LastName='Wo'
				//fmt.Println(n.macAddr, "AP의 fwVer이 변경되었습니다.. DB 값 수정합니다..", updateQuerySt)
				up, err := db.Prepare(updateQuerySt)
				if err != nil {
					panic(err)
				}
				defer up.Close()
				_, err = up.Exec()
				if err != nil {
					// the query, Scan returns ErrNoRows. 조건에 맞는 row가 없으면 프로그램이 종료되버림. ㅠ.ㅠ
					if err == sql.ErrNoRows {
						// there were no rows, but otherwise no error occurred
					//	fmt.Println("DB에 FW 쓰다가 에러 발생 -- [젬피]", sql.ErrNoRows)
					} else {
						fmt.Println("해당 FW의 row가 없습니다 !! -- [젬피]", sql.ErrNoRows)
						//log.Fatal(err)
					}
				}
			}
		if n.bootFlag==4&&s.flagFirm=="1"{
			updateQuerySt := "update " + st + " set flagFirm = \"2\" WHERE macAddr = '" + n.macAddr + "'" // ip를 AP에서 들어온 값으로 업데이트 한다
			// UPDATE Orders SET FirstName = 'John', LastName = 'Who' WHERE LastName='Wo'
			fmt.Println(n.macAddr, "AP의 설정 플래그가 변경되었습니다.. DB 값 수정합니다..", updateQuerySt)
			up, err := db.Prepare(updateQuerySt)
			if err != nil {
				panic(err)
			}
			defer up.Close()
			_, err = up.Exec()
			if err != nil {
				// the query, Scan returns ErrNoRows. 조건에 맞는 row가 없으면 프로그램이 종료되버림. ㅠ.ㅠ
				if err == sql.ErrNoRows {
					// there were no rows, but otherwise no error occurred
					fmt.Println("DB에 FW 쓰다가 에러 발생 -- [젬피]", sql.ErrNoRows)
				} else {
					fmt.Println("해당 FW의 row가 없습니다 !! -- [젬피]", sql.ErrNoRows)
					//log.Fatal(err)
				}
			}
		}
			if n.bootFlag==3&&s.flagConf=="1"{
				updateQuerySt := "update " + st + " set flagConf = \"2\" WHERE macAddr = '" + n.macAddr + "'" // ip를 AP에서 들어온 값으로 업데이트 한다
				// UPDATE Orders SET FirstName = 'John', LastName = 'Who' WHERE LastName='Wo'
				fmt.Println(n.macAddr, "AP의 펌웨어 플래그가 변경되었습니다.. DB 값 수정합니다..", updateQuerySt)
				up, err := db.Prepare(updateQuerySt)
				if err != nil {
					panic(err)
				}
				defer up.Close()
				_, err = up.Exec()
				if err != nil {
					// the query, Scan returns ErrNoRows. 조건에 맞는 row가 없으면 프로그램이 종료되버림. ㅠ.ㅠ
					if err == sql.ErrNoRows {
						// there were no rows, but otherwise no error occurred
						fmt.Println("DB에 FW 쓰다가 에러 발생 -- [젬피]", sql.ErrNoRows)
					} else {
						fmt.Println("해당 FW의 row가 없습니다 !! -- [젬피]", sql.ErrNoRows)
						//log.Fatal(err)
					}
				}
			}

	}

	}

	/******************	GET 화면 구성.. ********************/ // "\""  :: 따옴표 출력 하는 방법임
	querySt = "SELECT flagFirm, flagConf FROM " + st + " WHERE macAddr = " + "\"" + n.macAddr + "\""
	//fmt.Println(querySt)   // 앞으로 모두 이렇게 코딩하자... 그래야 디버깅 하기 쉽다
	err = db.QueryRow(querySt).Scan(&s.flagFirm, &s.flagConf)
	if err != nil {
		// the query, Scan returns ErrNoRows. 조건에 맞는 row가 없으면 프로그램이 종료되버림. ㅠ.ㅠ
		if err == sql.ErrNoRows { // there were no rows, but otherwise no error occurred	// 처음 수신되서 ap 테이블에 mac 이 없을 수 있다...
			fmt.Println("플래그 확인중... 해당하는 MAC 주소가 없습니다 (신규수신)", sql.ErrNoRows)
			w.Write([]byte("@@@ DB 쿼리 에러 -- 해당하는 MAC 주소가 없습니다\n"))
		} else {
			fmt.Println("업데이트 플래그 읽어오지 못했습니다 -- [젬피]", sql.ErrNoRows)
			//log.Fatal(err)
		}
	}
	// fmt.Println("------------------------->>>",  s.flagConf, s.flagFirm, s.macAddr)
	// 기본 값은 0 이다.  DB에서 해당 mac을 찾지 못하더라도 염려 없다. 기본이 zero
	flagFirm := s.flagFirm
	flagConf := s.flagConf
	// urlFirm := "www.umanager.kr:8100/abcd-1234.zip"
	// urlConf := "www.umanager.kr:8100/12345678.zip"
	urlFirm := url_of_Firm
	urlConf := url_of_Conf_75xx // 기본값은 MAP7500 임....

	var thisMacIs7500 string = "no" // 7500 인지 분간
	if modelName == "MAP7700" {     // 7700 인지 분류작업
		urlConf = url_of_Conf_77xx
	} else if modelName == "MAP9220" {
		urlConf = url_of_Conf_922x
	} else if modelName == "MAP7500" {
		thisMacIs7500 = "yes"
		urlConf = url_of_Conf_75xx
		modelName = "MAP7500"
	} else if modelName == "MAP5010" {
		urlConf = url_of_Conf_50xx
		modelName = "MAP5010"
	} else if modelName == "MAP2010" {
		urlConf = url_of_Conf_20xx
	} else if modelName == "MAP9200" {
		urlConf = url_of_Conf_920x
	}else if modelName == "5000_MESH"{
		urlConf = url_of_Conf_50xx_mesh
	}else{
		modelName = "MAP9200"
	}

	mac = strings.Replace(n.macAddr, ":", "", -1)
	macTag= mac[len(mac)-6:]

	println(macTag + "=" + n.ipAddr + " ") // 빨간 글씨로 찍어준다
	hashValue, err := ComputeMd5("file_storage/cf_" + macTag + ".zip")
	if err != nil {
		//fmt.Printf("Err: %v\n", err)
		hashValue = nil
	} else {
		if debug_level == "1" {
			fmt.Printf("The md5 checksum is: %x\n", hashValue)
			fmt.Println("ssid0: ", n.ssidW0, "ssid1: ", n.ssidW1)
		}
	}
	hashValue1, err := ComputeMd5("file_storage/root_uImage.MAP7700")
	if err != nil {
		fmt.Printf("Err: %v\n", err)
	} else {
		if debug_level == "1" {
			fmt.Printf("The md5 checksum is: %x\n", hashValue1)
			fmt.Println("ssid0: ", n.ssidW0, "ssid1: ", n.ssidW1)
		}
	}
	hashValue2, err := ComputeMd5("file_storage/bootpImage.map7500")
	if err != nil {
		fmt.Printf("Err: %v\n", err)
	} else {
		if debug_level == "1" {
			fmt.Printf("The md5 checksum is: %x\n", hashValue2)
			fmt.Println("ssid0: ", n.ssidW0, "ssid1: ", n.ssidW1)
		}
	}
	hashValue3, err := ComputeMd5("file_storage/root_uImage.MAP9220")
	if err != nil {
		fmt.Printf("Err: %v\n", err)
	} else {
		if debug_level == "1" {
			fmt.Printf("The md5 checksum is: %x\n", hashValue3)
			fmt.Println("ssid0: ", n.ssidW0, "ssid1: ", n.ssidW1)
		}
	}
	hashValue4, err := ComputeMd5("file_storage/root_uImage.MAP5010")
	if err != nil {
		fmt.Printf("Err: %v\n", err)
	} else {
		if debug_level == "1" {
			fmt.Printf("The md5 checksum is: %x\n", hashValue3)
			fmt.Println("ssid0: ", n.ssidW0, "ssid1: ", n.ssidW1)
		}
	}
	hashValue5, err := ComputeMd5("file_storage/root_uImage.MAP2010")
	if err != nil {
		fmt.Printf("Err: %v\n", err)
	} else {
		if debug_level == "1" {
			fmt.Printf("The md5 checksum is: %x\n", hashValue3)
			fmt.Println("ssid0: ", n.ssidW0, "ssid1: ", n.ssidW1)
		}
	}
	hashValue6, err := ComputeMd5("file_storage/root_uImage.MAP9200")
	if err != nil {
		fmt.Printf("Err: %v\n", err)
	} else {
		if debug_level == "1" {
			fmt.Printf("The md5 checksum is: %x\n", hashValue6)
			fmt.Println("ssid0: ", n.ssidW0, "ssid1: ", n.ssidW1)
		}
	}
	hashValue7, err := ComputeMd5("file_storage/root_uImage.5000_MESH")
	if err != nil {
		fmt.Printf("Err: %v\n", err)
	} else {
		if debug_level == "1" {
			fmt.Printf("The md5 checksum is: %x\n", hashValue7)
			fmt.Println("ssid0: ", n.ssidW0, "ssid1: ", n.ssidW1)
		}
	}
	var stringHashValue string // config 설정 하는게 아니면 물결무니만 나오게 한다 얘는 컨피그
	if flagFirm == "1" {       // 7500 일경우 컨피그 업데이트 안하더라도 컨피그 파일하고 해쉬값 있어야 한다
		if FileExists("file_storage/cf_" + macTag + ".zip") {
			stringHashValue = fmt.Sprintf("%x", hashValue)
		}
	}

	var stringHashValue1 string
	var stringHashValue2 string // config 설정 하는게 아니면 물결무니만 나오게 한다  얘는 펌
	var stringHashValue3 string
	var stringHashValue4 string
	var stringHashValue5 string
	var stringHashValue6 string
	var stringHashValue7 string
	if flagConf == "1" { // 7500 일경우 컨피그 업데이트 안하더라도 컨피그 파일하고 해쉬값 있어야 한다
		if thisMacIs7500 == "yes" {
			if FileExists("file_storage/bootpImage.map7500") {
				stringHashValue2 = fmt.Sprintf("%x", hashValue2)
			} else {
				stringHashValue2 = fmt.Sprintf("%x", "nofile")
			}
		} else if modelName == "MAP7700" {
			if FileExists("file_storage/root_uImage.MAP7700") {
				stringHashValue1 = fmt.Sprintf("%x", hashValue1)
			} else {
				stringHashValue1 = fmt.Sprintf("%x", "nofile")
			}
		} else if modelName == "MAP9220" {
			if FileExists("file_storage/root_uImage.MAP9220") {
				stringHashValue3 = fmt.Sprintf("%x", hashValue3)
			} else {
				stringHashValue3 = fmt.Sprintf("%x", "nofile")
			}
		} else if modelName == "MAP5010" {
			if FileExists("file_storage/root_uImage.MAP5010") {
				stringHashValue4 = fmt.Sprintf("%x", hashValue4)
			} else {
				stringHashValue4 = fmt.Sprintf("%x", "nofile")
			}
		} else if modelName == "MAP2010" {
			if FileExists("file_storage/root_uImage.MAP2010") {
				stringHashValue5 = fmt.Sprintf("%x", hashValue5)
			} else {
				stringHashValue5 = fmt.Sprintf("%x", "nofile")
			}
		} else if modelName == "MAP9200" {
			if FileExists("file_storage/root_uImage.MAP9200") {
				stringHashValue6 = fmt.Sprintf("%x", hashValue6)
			} else {
				stringHashValue6 = fmt.Sprintf("%x", "nofile")
			}
		}else if modelName == "5000_MESH" {
			if FileExists("file_storage/root_uImage.5000_MESH") {
				stringHashValue7 = fmt.Sprintf("%x", hashValue7)
			} else {
				stringHashValue7 = fmt.Sprintf("%x", "nofile")
			}
		}
	}
	fmt.Println(modelName)
	// 가급적 웹방식 http 프로토콜과 포트 80 사용하여 방화벽 차단을 최소화 한다
	if modelName == "MAP7700" {
		w.Write([]byte(flagFirm + " " + flagConf + " " + urlFirm + " " + stringHashValue + " ~~~~ " + urlConf + " " + stringHashValue1))
		w.Write([]byte("\r\n____________________________________________________\r\n"))
	} else if modelName == "MAP7500" {
		w.Write([]byte(flagFirm + " " + flagConf + " " + urlFirm + " " + stringHashValue + " ~~~~ " + urlConf + " " + stringHashValue2))
		w.Write([]byte("\r\n____________________________________________________\r\n"))
	} else if modelName == "MAP9220" {
		w.Write([]byte(flagFirm + " " + flagConf + " " + urlFirm + " " + stringHashValue + " ~~~~ " + urlConf + " " + stringHashValue3))
		w.Write([]byte("\r\n____________________________________________________\r\n"))
	} else if modelName == "MAP5010" {
		w.Write([]byte(flagFirm + " " + flagConf + " " + urlFirm + " " + stringHashValue + " ~~~~ " + urlConf + " " + stringHashValue4))
		w.Write([]byte("\r\n____________________________________________________\r\n"))
	} else if modelName == "MAP2010" {
		w.Write([]byte(flagFirm + " " + flagConf + " " + urlFirm + " " + stringHashValue + " ~~~~ " + urlConf + " " + stringHashValue5))
		w.Write([]byte("\r\n____________________________________________________\r\n"))
	} else if modelName == "MAP9200" {
		w.Write([]byte(flagFirm + " " + flagConf + " " + urlFirm + " " + stringHashValue + " ~~~~ " + urlConf + " " + stringHashValue6))
		w.Write([]byte("\r\n____________________________________________________\r\n"))
	}else if modelName == "5000_MESH" {
		w.Write([]byte(flagFirm + " " + flagConf + " " + urlFirm + " " + stringHashValue + " ~~~~ " + urlConf + " " + stringHashValue7))
		w.Write([]byte("\r\n____________________________________________________\r\n"))
	}
	if queryError {
		fmt.Println("[ERROR] DB 컬럼명과 다른 문자로 쿼리가 들어 왔습니다")
	}

	// 루프 돈 횟수와 파싱한 내용을 웹브라우져에 보내준다
	w.Write([]byte("총 키 갯수는 " + strconv.Itoa(i) + " 개 입니다\n" + "받은 쿼리 정리합니다\n\n"))
	w.Write([]byte(returnString))
	if debug_level == "1" {
		print(inputQueryString)
	}
	inputQueryString = ""

	if magicFlag != true {
		w.Write([]byte("\r\n[ERROR] 매직넘버 오류..\n\n\n"))
		//return
	}
	/*****************	쿼리 DB 입력	*****************/
	// site 테이블에 mac이 없어도 쿼리 들어오면 무조건 테이블에 인서트한다...
	// 위에 설정파일때문에 열었으니까 아래 디비 여는건 생략해도 되지 않나??? 나중에 통편집해서 확인 필요.
	db, err = sql.Open("mysql", "root:1111@tcp("+dbLocation+":"+db_port+")/map")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	if newmonthap>10{
		newmonthap=0;
	}
	if newmonthtable>10{
		newmonthtable=0;
	}
	//print(newmonthap,newmonthtable)
	st = "ap" + curYearMonth // ""AP"" 테이블명 재확정
	dbrow = st
	stmtIns, err := db.Prepare("INSERT INTO " + st + " " + apTableInsert) // 테이블명 + 집어넣는 필드와 플레이스 홀더. INTO 뒤에 그리고 st 다음에도 공백하나 넣어라..
	if err != nil {
		timenow := time.Now().String()
		f,_ = os.OpenFile("./log/"+timenow[0:7]+"-stoplog.csv", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		fmt.Println("TalkingAP DB insert 준비 하는데서 에러발생!!")
		newmonthtable++
		if newmonthtable == 1 {
			go open("http://" + db_url + ":8080/maketable?yearmonth=" + curYearMonth)

		}
		//log.Fatal(err)
	} else {
		newmonthtable = 0
	}
	//db에 넣는걸 Prepare하고 그다음에 그 프리페어가 저장된 변수.Exec해서 진짜 값 넣는거.
	//그니까 미리 테이블하고 변수 ? 해놓고 ?에 변수exec해서 value넣으면 쏙 들어간다.
	defer stmtIns.Close()

	n.time = uint32(time.Now().Unix()) // 현재시각 받기

	//  오타와의 싸움..
	_, err = stmtIns.Exec(n.id, n.time, n.bootFlag, n.macAddr, n.ipAddr, n.btVer, n.fwVer, n.ssidW0, n.ssidW1,
		n.chanW0, n.chanW1, n.rxByteW0, n.txByteW0, n.rxByteW1, n.txByteW1, n.rxPktW0, n.txPktW0, n.rxPktW1, n.txPktW1, n.assoCtW0,
		n.assoCtW1, n.devTemp, n.curMem, n.pingTime, n.assoDevT, n.leaveDevT, n.peerAp0, n.peerAp1, n.staMac0) //

	checkError(err)

	// 양대 테이블 해당 월의 ROW 갯수 확인 하기
	/*
		//var siteTableCnt int
		var apTableCnt int
		//disk & cpu 부하줄이기 운동
		err = db.QueryRow("SELECT COUNT(*) FROM " + st).Scan(&apTableCnt)
		fmt.Println(st + " 테이블의 Row: ", apTableCnt)

	*/
	/*
		st = "site" + curYearMonth // ""site"" 테이블명으로 바꿔서 --> 테이블 row 갯수가 간단하면서도 젤 중요한 정보임
		err = db.QueryRow("SELECT COUNT(*) FROM " + st).Scan(&siteTableCnt)
		fmt.Println(st + " 테이블의 Row: ", siteTableCnt)
	*/
	//////////// 웹소켓으로 관리패킷 전달해야지.. 받는게 아님..
	fakemac=n.macAddr
	gMsgSentCnt++
	fmt.Println(meshresultstring)
	mutex.Lock()
	for _, meshmapre := range meshmapstring {
		meshresultstring+=meshmapre
	}

	mutex.Unlock()

}
func writeTr(mac string,tx uint64,rx uint64){
	timenow := time.Now().String()
	var trudata string
	var b string
	//var txx,rxx uint64
	a,_:=ioutil.ReadFile("./log/"+timenow[0:7]+"-datatrafic.csv")
	if(strings.Contains(string(a),mac)){
	}else{
		b+=fmt.Sprint(string(a)+timenow[0:7], ",Mac=", mac, ",tx=", tx, ",rx=", rx, "\n")
		ioutil.WriteFile("./log/" + timenow[0:7] + "-datatrafic.csv",[]byte(b),os.FileMode(644))
	}
	traex, err := readLines("./log/" + timenow[0:7] + "-datatrafic.csv")
	if err != nil {
		fmt.Println(err)
	}
	if traex == nil {
		fmt.Println("비어있다.")
	} else {
		//fmt.Println(traex)
		for _, line := range traex {
			if line != "" && strings.Contains(line, mac) {
				s := strings.Split(line, ",")
				//fmt.Println(s[0], "!!!")
				mac:= strings.Split(s[1], "=")
				//txx, _ = strconv.ParseUint(txtra[1], 10, 64) // 10 진수로 int64 로 변환하기
				//rxx, _ = strconv.ParseUint(rxtra[1], 10, 64) // 10 진수로 int64 로 변환하기
				//fmt.Println(mac[1], txx+tx, rxx+rx)
				trudata+=fmt.Sprint(timenow[0:7], ",Mac=", mac[1], ",tx=", tx, ",rx=", rx, "\n")
			}else{
				trudata+=line+"\n"
			}
		}
		ioutil.WriteFile("./log/" + timenow[0:7] + "-datatrafic.csv",[]byte(trudata),os.FileMode(644))
		//fmt.Println(trudata)
	}
}
func readTr(mac string, rx uint64, tx uint64)(uint64, uint64, uint64, uint64){
	var txxx,rxxx uint64
	aq,err:=ioutil.ReadFile("./tmp/"+mac)
	if err!=nil{
		os.Mkdir("./tmp/",os.FileMode(644))
		fmt.Println(err)
		ioutil.WriteFile("./tmp/"+mac,[]byte(fmt.Sprint(rx,"/",tx,"/0/0")),os.FileMode(644))
		return 0,0,0,0
	}else{
		if strings.Count(string(aq),"/")!=3{
			ioutil.WriteFile("./tmp/"+mac,[]byte(fmt.Sprint("0/0/",rx,"/",tx)),os.FileMode(644))
			return 0,0,0,0
		}else{
			a:=strings.Split(string(aq),"/")
			rxx, err := strconv.ParseUint(a[0], 10, 64) // 10 진수로 int64 로 변환하기
			if err!=nil{
				rxx=0
			}
			txx, err := strconv.ParseUint(a[1], 10, 64) // 10 진수로 int64 로 변환하기
			if err!=nil{
				txx=0
			}
			rxtotal, err := strconv.ParseUint(a[2], 10, 64) // 10 진수로 int64 로 변환하기
			if err!=nil{
				rxtotal=0
			}
			txtotal, err := strconv.ParseUint(a[3], 10, 64) // 10 진수로 int64 로 변환하기
			if err!=nil{
				txtotal=0
			}
			if txx>tx{
				//fmt.Println("저장된 tx가 크다")
				txxx=txx
			}else{
				txxx=tx-txx
			}
			if rxx>rx{
				//fmt.Println("저장된 rx가 크다")
				rxxx=rxx
			}else{
				rxxx=rx-rxx
			}
			//fmt.Println(txtotal,rxtotal)
			ioutil.WriteFile("./tmp/"+mac,[]byte(fmt.Sprint(rx,"/",tx,"/",rxtotal+rxxx,"/",txtotal+txxx)),os.FileMode(644))
			return txxx,rxxx,txtotal+txxx,rxtotal+rxxx
		}

	}
}
func loadconfig(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Cache-Control", "no-cache, private, max-age=0")
	w.Header().Set("Expires", time.Unix(0, 0).Format(http.TimeFormat))
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Accel-Expires", "0")
	var ip,message,filestring string;
	r.ParseForm()
	macip := r.Form.Get("macaddr")
	mac := strings.Replace(macip, ":", "", -1)
	macTag := mac[len(mac)-6:]
	fmt.Println(macip)
	db, err := sql.Open("mysql", "root:1111@tcp("+dbLocation+":"+db_port+")/map")
	//rw, err:=db.Query("SHOW TABLE STATUS FROM map")
	//fmt.Println("11111111111111",rw,err,"aaaaasdasdoqwojeliahsdlh")
	if err != nil {
		panic(err)
	}
	curYearMonth := time.Now().String()
	curYearBuf := curYearMonth[:4]                 // 연도 짤라내기. 글자수는 0부터 시작하고 끝 숫자는 빠진다. 0,1,2,3 위치 잘라내기
	curYearMonth = curYearBuf + curYearMonth[5:7]
	site := "site" + curYearMonth
	//n := new(siteApTableVal)
	mactoweb, err := db.Query("SELECT ipAddr FROM " + site + " where macAddr='" + macip + "'")
	for mactoweb.Next() {
		err := mactoweb.Scan(&ip)
		if err != nil {
			fmt.Println("\n사이트 테이블 읽어오는데서 에러 발생\n")
			//log.Fatal(err)
		}
	}
	defer db.Close()
	datasplit:=":SPLIT:"
	stringsplit:="SPLITNOW"
	port:="4000"
	// connect to this socket
	conn, err := net.Dial("tcp", ip+":"+port)
	if err!=nil{
		w.Write([]byte("No response From "+ip))
		fmt.Println("서버가 ㅈ구ㅇ어있다")
	}else{
		defer conn.Close()
		// read in input from stdin
		text:="Send Config Plz"
		fmt.Println(text)
		// send to socket
		fmt.Fprintf(conn, text)
		// listen for reply
		//conn.Read(buffer)
		message, _ = bufio.NewReader(conn).ReadString('/')
	}
	if strings.Contains(message,datasplit){
		a:=strings.Split(message,datasplit)
		inic:=strings.Split(a[0],stringsplit)
		os.Mkdir("./file_storage/cf_"+macTag+"/",os.FileMode(644))
		err=ioutil.WriteFile("./file_storage/cf_"+macTag+"/"+inic[1],[]byte(inic[0]),os.FileMode(644))
		rt:=strings.Split(a[1],stringsplit)
		test:=strings.SplitN(rt[0],"\n",-1)
		for i,_:=range test{
			if test[i]=="'"{
				test[i-1]=test[i-1]+test[i];
				test[i]=""
			}
		}
		for i,_:=range test{
			if len(test[i])<2{
			}else{
				filestring+=test[i]+"\n"
			}
		}
		fmt.Println(macTag)
		filestring="#The following line must not be removed.\nDefault\n"+filestring
		//os.Mkdir("./file_storage/cf_"+macTag+"/"+rt[1],os.FileMode(644))
		err=ioutil.WriteFile("./file_storage/cf_"+macTag+"/"+rt[1],[]byte(filestring),os.FileMode(644))
		if err!=nil{
			fmt.Println("파일 쓰는데 이상이 생김. ")
			fmt.Println(err)
		}
		conf_table :=
			"<!DOCTYPE html> <html>	<html lang=\"en\"> <head><style>#wrapper{border: 2px solid blue;width:100%;}#wrapper2{border: 2px solid blue;width:100%;}</style>" + "<meta charset=\"UTF-8\"> </head> <body> <div id='wrapper'><div style='background-color:black;color:white'>JMP SYSTEMS Co., Ltd. 2017</div>"
		conf_table += "</div><div id='wrapper21' style='height:100px'>"+macip+"<input type='button' id='save' value='불러온 설정파일 저장하기'style='float:right'onclick='thisis();'><textarea style= 'min-width:100%; max-width:100%; min-height:480px; max-height:480px'>"+inic[0]+filestring+"</textarea><input type='button'value='불러온 설정파일 저장하기'style='float:right'onclick='thisis()'></body><script>var mac='"+macip+"';function thisis(){window.open('exchange?macaddr="+macip+"','_self');}function LockF5() {if(event.ctrlKey&&event.keyCode==83){var button = document.getElementById('save');button.click();return false}console.log(event.keyCode);}document.onkeydown = LockF5;</script> </html>"

		w.Write([]byte(conf_table))
		//a:=strings.Split(a[1],"SPLITNOW")
	}else{
		w.Write([]byte(message))
		fmt.Println(message)
	}
}
func FileExists(name string) bool {

	if _, err := os.Stat(name); os.IsNotExist(err) {
		return false
	}
	return true
}
var maxhop = 0;
func checkmeshflag2(meshmac string)int{
	var Firm,Conf string
	curYearMonth := time.Now().String()
	curYearBuf := curYearMonth[:4]                // 연도 짤라내기. 글자수는 0부터 시작하고 끝 숫자는 빠진다. 0,1,2,3 위치 잘라내기
	curYearMonth = curYearBuf + curYearMonth[5:7] // 월 짤라내기
	st := "site" + curYearMonth // ""사이트"" 테이블명 확정
	db, err := sql.Open("mysql", "root:1111@tcp("+dbLocation+":"+db_port+")/map")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	meshuparray:=strings.SplitN(meshmac,"/",-1)
	for i:=0;i<len(meshuparray)-1;i++{
		//fmt.Println(strings.Split(meshuparray[i],",")[0])
		macstring :=strings.Split(meshuparray[i],",")[0]
		fmt.Println(macstring)
		queryStr := "SELECT flagFirm, flagConf FROM " + st + " WHERE macAddr = " + "\"" + macstring + "\""
		err = db.QueryRow(queryStr).Scan(&Firm,&Conf)
		if Conf=="0"&&Firm!="2"{
			return -1
		} else if Firm=="0"&&Conf!="2"{
			return -1
		}
		//queryst:= "update "+st+" set "+qrstring+" where macAddr = \"" + macstring+"\""

	}
	return 0
}
func setmeshflag(mac string,flag string){

	meshflagmap = make(map[int]string)
	//fmt.Println(hopcount[mac])
	fmt.Println("맥넣으면 순서   bookmark : ",hopcount)
	fmt.Println("맥이랑 플래그 bookmark : ",mac)
	meshuparray:=strings.SplitN(mac,"/",-1)
	for i:=0;i<len(meshuparray)-1;i++{
		//fmt.Println(strings.Split(meshuparray[i],",")[0])
		meshflagmap[hopcount[strings.Split(meshuparray[i],",")[0]]]+=meshuparray[i]+";"
		if hopcount[strings.Split(meshuparray[i],",")[0]]>=maxhop{
			maxhop=hopcount[strings.Split(meshuparray[i],",")[0]]
		}
	}
	if flag=="2"{
		sethopmeshflag(maxhop,"2")
	}
	/*for key, val := range hopcount {
		meshflagmap[val]+=key+","
		if(val>=maxhop){
			maxhop=val;
		}
		//fmt.Println(key, val)
	}*/
	//fmt.Println(maxhop)
	//sethopmeshflag(maxhop,meshconfw)
	//fmt.Println(maxhop)
	//fmt.Println(meshflagmap)
}
func sethopmeshflag(i int,flag string){
	qrstring:=""
	if flag == "1" {
		qrstring = "flagConf = \"3\""
	}else if flag=="2" {
		qrstring = "flagFirm = \"1\""
	}
	curYearMonth := time.Now().String()
	curYearBuf := curYearMonth[:4]                 // 연도 짤라내기. 글자수는 0부터 시작하고 끝 숫자는 빠진다. 0,1,2,3 위치 잘라내기
	curYearMonth = curYearBuf + curYearMonth[5:7]
	st := "site" + curYearMonth
	fmt.Println(meshflagmap[i],"이거")
	meshfwupdateresult1:=strings.SplitN(meshflagmap[i],";",-1)
	fmt.Println("홉 수",maxhop," / <--에 해당하는 갯수",len(meshfwupdateresult1)-1)
	if len(meshfwupdateresult1)-1==0{
		if maxhop==0{
			return
		}
		maxhop--
		sethopmeshflag(maxhop,flag)
		return
	}
	meshfwupdateresult=len(meshfwupdateresult1)-1
	fmt.Println(meshfwupdateresult1)
	db, err := sql.Open("mysql", "root:1111@tcp("+dbLocation+":"+db_port+")/map")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	//querySt := "update " + st + " set flagFirm = \"0\", flagConf = \"0\"" // 아진짜.
	for k:=0;k<len(meshfwupdateresult1)-1;k++{
		meshdbstring := strings.SplitN(meshfwupdateresult1[k],",",-1)
		queryst:= "update "+st+" set "+qrstring+" where macAddr = \"" + meshdbstring[0]+"\""
		//fmt.Println(queryst)
		up, err := db.Prepare(queryst)
		if err != nil {
			panic(err)
		}
		defer up.Close()
		_, err = up.Exec()
		if err != nil {
			// the query, Scan returns ErrNoRows. 조건에 맞는 row가 없으면 프로그램이 종료되버림. ㅠ.ㅠ
			if err == sql.ErrNoRows {
				// there were no rows, but otherwise no error occurred
				fmt.Println("<플래그 세팅> DB 에러 발생 -- [젬피]", sql.ErrNoRows)
			} else {
				fmt.Println("<플래그 세팅> 해당 MAC 주소의 row가 없습니다 !! -- [젬피]", sql.ErrNoRows)
				//log.Fatal(err)
			}
		}
	}

	//strings.Split(meshflagmap,",")
}
func setFlagAll(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache, private, max-age=0")
	w.Header().Set("Expires", time.Unix(0, 0).Format(http.TimeFormat))
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Accel-Expires", "0")

	w.Write([]byte("해당 MAC AP의 플래그를 1로 설정하려고 합니다만...\n\n설정 적용시 1분, 펌업은 5분 소요됩니다.\n\n진행상황 ::::::::::::::::>>>"))

	// 여기서 팝업창 내용에 해당하는 html 콘텐츠를 밀어 넣어 준다... 그리고 슬립시켜가면서 기다리는 상태 표출해준다. 장비로 핑나가면 그때서 알려준다.
	// 디자인은 x 패널에 내부는 테이블 스타일에 프로그래스바 진행되도록....
	r.ParseForm()
	log.Println(r.Form)
	// DB 인서트할 내용 모음 : 구문용
	var st string
	// DB 인서트할 내용 모음 : 플레이스 홀더에 넣을 변수(일단은 모두 스트링으로 모은다)
	var ph string
	// 쿼리 에러 메시지를 나중에 표출하기 위한 플래그
	var queryError bool
	// 쿼리 받아서 site DB 테이블에 값을 구조체 변수 생성
	s := new(siteApTableVal)
	// 브라우저에 리턴해줄 글 모음
	var returnString string
	var multiMacAddrHanddleCnt int
	var macAddrBuf [100]string // 동시 100대 까지 업그레이드 처리 가능
	// 쿼리 추출 루프 도는 횟수 카운팅
	//var i int
	//	var magicFlag = false
	// map 이라는 것은 unordered 이므로 "순서는 무작위"로 나오게 된다. 키+값 조합 갯수는 정확
	// map을 for 문에서 사용할때 항상 k, v 두가지 변수를 쓰고 이 변수는 map 변수를 따라가게된다. 덕타이핑~~
	// key가 필요 없으면 _ 으로 처리한다
	for k, v := range r.Form {
		// 매직넘버 검사. 위치가 랜덤이므로 전수검사한다.
		if k == "magic" {
			// fmt.Println("매직넘버 처리 루틴 들어옴")
			if strings.Join(v, "") == "415$$" {
				//magicFlag = true  // 올바른 매직넘버 존재 확인 됨
				// 이것은 DB에 저장할 필요가 없으므로 뺀다. 콘티뉴 신공
				continue
			}
		}
		// 아래는 브라우저에 보여주기 위함이고 이게 진짜 중요함
		tempV := strings.Join(v, "") // 문자 map value 값을 string으로 변환하기

		switch k {
		case "macAddr": // url 값 표준을 따라가자면 js 올때부터 결국 macAddr 이란 키 하나에 value들은 모두 묶여서 날라온다.. 그것도 공백없이..
			// map[flagFirm:[1] flagConf:[0] example_length:[10] select_all:[1] macAddr:[00:06:7a:02:03:87 00:06:7a:20:20:84]]
			s.macAddr = tempV
		case "flagFirm":
			s.flagFirm = tempV
		case "flagConf":
			s.flagConf = tempV
		case "example_legnth":
		case "select_all":
		default:
			queryError = true
		}
		fmt.Println(s.flagFirm)
		st += k + ","
		ph += strings.Join(v, "") + "," // map 내에 있는 값은 []string 으로 정의되어 있어 string과는 호환되지 않음.
		// 그리고 키값 하나에 하나의 []string 밖에 없으므로 조인해도 하나만 나온다. 여러게 있어야 Join 내부의 ""안 값이 의미를 가진다.. 결과적으로 ","을 뒤에 추가하여 csv 형식을 만든다.
		returnString += "key: " + k + "\n"
		returnString += "val: " + strings.Join(v, "") + "\n"

	}
	for i := 0; i < len(s.macAddr)/17; i++ {
		macAddrBuf[i] = s.macAddr[17*i : 17*i+17] //  mac 문자 길이 17 자
	//	fmt.Println("============= ", len(s.macAddr), s.macAddr, i, macAddrBuf[i])
		multiMacAddrHanddleCnt++
	}

	if queryError {
		fmt.Println("지정된 url 요소외 다른 내용이 입력되었습니다. MAC이나 flag 값이 없을 수도 있습니다\n")
	}
	curYearMonth := time.Now().String()
	curYearBuf := curYearMonth[:4]                // 연도 짤라내기. 글자수는 0부터 시작하고 끝 숫자는 빠진다. 0,1,2,3 위치 잘라내기
	curYearMonth = curYearBuf + curYearMonth[5:7] // 월 짤라내기

	st = "site" + curYearMonth // ""사이트"" 테이블명 확정

	/******************	설정값 변경 지시	********************/
	//디비에서 읽어와서 결정한다
	db, err := sql.Open("mysql", "root:1111@tcp("+dbLocation+":"+db_port+")/map")
	if err != nil {
		panic(err)
	}
	meshmacstring=""
	for i := 0; i < multiMacAddrHanddleCnt; i++ {

	}

	defer db.Close()
	// update map.site201612 set flagFirm = "0", flagConf = "0" where macAddr = "00:06:7a:e7:20:88";
	if macAddrBuf[0] == "" { // 최소한 1개 라도 맥이 있어야 함
		fmt.Println("\n<플래그 세팅> MAC 주소가 없습니다. 쿼리로 보내주세요 :", s.flagConf, s.flagFirm, s.macAddr)
		return
	}
	if macAddrBuf[0] == "all" { // 맥주소 값이 all 일 경우는 MAC 주소 상관없이 모두 1로 세팅함.
		// /*******************  그냥 macAddr 만 all로 보내주면 되니까.. 이건 자바쪽 첵빡과 무관하게 별도 버튼으로 만들어 날려주자 *************************//
		querySt := "update " + st + " set flagFirm = \"" + s.flagFirm + "\", flagConf = \"" + s.flagConf + "\"" // 아진짜.
		//  쿼리에서 주는값 그대로 DB에 옮긴다
		fmt.Println(querySt)
		up, err := db.Prepare(querySt)
		if err != nil {
			panic(err)
		}
		defer up.Close()
		_, err = up.Exec()
		if err != nil {
			// the query, Scan returns ErrNoRows. 조건에 맞는 row가 없으면 프로그램이 종료되버림. ㅠ.ㅠ
			if err == sql.ErrNoRows {
				// there were no rows, but otherwise no error occurred
				fmt.Println("DB 에러 발생 -- [젬피]", sql.ErrNoRows)
			} else {
				fmt.Println("해당 MAC 주소의 row가 없습니다 !! -- [젬피]", sql.ErrNoRows)
				//log.Fatal(err)
			}
		}
	} else {
		for i := 0; i < multiMacAddrHanddleCnt; i++ { // 다중 맥 플래그 세팅 해준다
			var checkmesh string
			queryStr := "SELECT modelName FROM " + st + " WHERE macAddr = " + "\"" + macAddrBuf[i] + "\""
			err = db.QueryRow(queryStr).Scan(&checkmesh)
			if strings.Contains(checkmesh,"MESH"){
				/*if meshfwupdateresult!=0{
					w.Write([]byte("\n이미 mesh 펌웨어 업데이트가 진행되고 있습니다."))
					return
				}else if meshconfw!="0"{
					w.Write([]byte("\n이미 mesh 펌웨어 업데이트 혹은 설정 변경이 진행되고 있습니다."))
				}*/
				if strings.Contains(meshmacstring,macAddrBuf[i])!=true{
					if s.flagFirm=="1"{
						meshconfw="2"
						meshmacstring+=macAddrBuf[i]+",2/"
					}else if s.flagConf=="1"{
						meshconfw="1"
						meshmacstring+=macAddrBuf[i]+",1/"
					}else{
						meshconfw="0"
						meshmacstring+=macAddrBuf[i]+",0/"

					}
				}

				//meshmacstring+=macAddrBuf[i]+","+s.flagFirm+","+s.flagConf+"/"
			}
			querySt:=""
			if strings.Contains(meshmacstring,macAddrBuf[i])!=true{
				querySt = "update " + st + " set flagFirm = \"" + s.flagFirm + "\", flagConf = \"" + s.flagConf + "\" WHERE macAddr = " + "\"" + macAddrBuf[i] + "\""
			}else{
				querySt = "update " + st + " set flagFirm = \"0\", flagConf = \"" + s.flagConf + "\" WHERE macAddr = " + "\"" + macAddrBuf[i] + "\""
			}

				fmt.Println(querySt)
				up, err := db.Prepare(querySt)
				if err != nil {
					panic(err)
				}
				defer up.Close()
				_, err = up.Exec()
				if err != nil {
					// the query, Scan returns ErrNoRows. 조건에 맞는 row가 없으면 프로그램이 종료되버림. ㅠ.ㅠ
					if err == sql.ErrNoRows {
						// there were no rows, but otherwise no error occurred
						fmt.Println("<플래그 세팅> DB 에러 발생 -- [젬피]", sql.ErrNoRows)
					} else {
						fmt.Println("<플래그 세팅> 해당 MAC 주소의 row가 없습니다 !! -- [젬피]", sql.ErrNoRows)
						//log.Fatal(err)
					}
			}
		}
		////qweqwe
		setmeshflag(meshmacstring,meshconfw)
		//fmt.Println("//////",meshmacstring)
	}

	//fmt.Println("------------------------->>>",  s.flagConf, s.flagFirm, s.macAddr) // 플래그 값 확인
}
func writing(w http.ResponseWriter, r *http.Request){
	k:=0
	j:=0;

	r.ParseForm()
	fmt.Println(r)
	writepars:=r.Form.Get("flag")
	var Firm,Conf string
	curYearMonth := time.Now().String()
	curYearBuf := curYearMonth[:4]                // 연도 짤라내기. 글자수는 0부터 시작하고 끝 숫자는 빠진다. 0,1,2,3 위치 잘라내기
	curYearMonth = curYearBuf + curYearMonth[5:7] // 월 짤라내기
	st := "site" + curYearMonth // ""사이트"" 테이블명 확정
	db, err := sql.Open("mysql", "root:1111@tcp("+dbLocation+":"+db_port+")/map")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	meshuparray:=strings.SplitN(meshmacstring,"/",-1)
	for i:=0;i<len(meshuparray)-1;i++{
		//fmt.Println(strings.Split(meshuparray[i],",")[0])
		macstring :=strings.Split(meshuparray[i],",")[0]
		fmt.Println(macstring)
		if writepars == "1"{
			queryStr := "SELECT flagConf FROM " + st + " WHERE macAddr = " + "\"" + macstring + "\""
			err = db.QueryRow(queryStr).Scan(&Conf)
			fmt.Println(Conf,"펌")
			if Conf!="2"{
				k++
			}
		}else if writepars =="2"{
			queryStr := "SELECT flagFirm FROM " + st + " WHERE macAddr = " + "\"" + macstring + "\""
			err = db.QueryRow(queryStr).Scan(&Firm)
			fmt.Println(Firm,"펌플래그")
			if Firm!="2"{
				j++
			}
		}
		//queryst:= "update "+st+" set "+qrstring+" where macAddr = \"" + macstring+"\""
	}
	if len(meshuparray)-1==0{
		w.Write([]byte("아직 펌웨어 업로드가 완료되지 않았거나 업로드된 펌웨어가 없습니다.."))
		return
	}
	fmt.Println(k,j,maxhop,"아이제이",len(meshuparray))
	if writepars =="1"{
		if k!=0{
			w.Write([]byte("아직 펌웨어 업로드가 완료되지 않았거나 업로드된 펌웨어가 없습니다.."))
		}else{
			sethopmeshflag(maxhop,"1")
			w.Write([]byte("펌웨어 업데이트를 시작합니다. \n 장비의 전원을 절대 끄지 마세요."))

		}
	}/*else if writepars=="2"{
		if j!=0||len(meshuparray)-1==0 {
			w.Write([]byte("아직 설정 파일 업로드가 완료되지 않았거나 업로드된 설정파일이 없습니다.."))
		}else{
			w.Write([]byte("설정파일 업데이트를 시작합니다. \n 장비의 전원을 절대 끄지 마세요."))
			sethopmeshflag(maxhop,"2")
		}
	}*/
}

func resetFlag(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Cache-Control", "no-cache, private, max-age=0")
	w.Header().Set("Expires", time.Unix(0, 0).Format(http.TimeFormat))
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Accel-Expires", "0")
	fmt.Println("장비의 Flag를 reset합니다.")

	w.Write([]byte("업데이트 플레그들을 리셋 합니다\n"))

	r.ParseForm()
	// DB 인서트할 내용 모음 : 구문용
	var st string
	// DB 인서트할 내용 모음 : 플레이스 홀더에 넣을 변수(일단은 모두 스트링으로 모은다)
	var ph string

	// 쿼리 에러 메시지를 나중에 표출하기 위한 플래그
	//	var queryError bool

	// 쿼리 받아서 DB 테이블에 값 넣을 구조체 변수 생성
	n := new(apTableVal)

	// 브라우저에 리턴해줄 글 모음
	var returnString string
	// 쿼리 추출 루프 도는 횟수 카운팅
	//var i int

	//var magicFlag = false

	// map 이라는 것은 unordered 이므로 "순서는 무작위"로 나오게 된다. 키+값 조합 갯수는 정확
	// map을 for 문에서 사용할때 항상 k, v 두가지 변수를 쓰고 이 변수는 map 변수를 따라가게된다. 덕타이핑~~
	// key가 필요 없으면 _ 으로 처리한다
	for k, v := range r.Form {

		// 매직넘버 검사. 위치가 랜덤이므로 전수검사한다.
		if k == "magic" {
			// fmt.Println("매직넘버 처리 루틴 들어옴")
			if strings.Join(v, "") == "415$$" {
				//	magicFlag = true  // 올바른 매직넘버 존재 확인 됨
				// 이것은 DB에 저장할 필요가 없으므로 뺀다. 콘티뉴 신공
				continue
			}
		}
		// 아래는 브라우저에 보여주기 위함이고 이게 진짜 중요함
		tempV := strings.Join(v, "") // 문자 map value 값을 string으로 변환하기
		switch k {
		case "macAddr":
			n.macAddr = tempV
		case "ipAddr":
			n.ipAddr = tempV
		case "upProg":
			gProg = tempV
		default:
			//			queryError = true
		}

		st += k + ","
		ph += strings.Join(v, "") + "," // map 내에 있는 값은 []string 으로 정의되어 있어 string과는 호환되지 않음. 그리고 키값 하나에 하나의 []string 밖에 없으므로 조인해도 하나만 나온다. 여러게 있어야 Join 내부의 ""안 값이 의미를 가진다.. 결과적으로 ","을 뒤에 추가하여 csv 형식을 만든다.

		//print("==========>", ph, "*******>", gProg)
		returnString += "key: " + k + "\n"
		returnString += "val: " + strings.Join(v, "") + "\n"
	}

	/*	if (!magicFlag) {
			fmt.Println("플래그 지우기에서 에러발생\n")
			return
		}

		if (queryError) {
			fmt.Println("수동으로 플래그를 clear 하려고 하였으나 찾아야할 MAC 주소가 없습니다.\n")
			return
		}
	*/
	curYearMonth := time.Now().String()
	curYearBuf := curYearMonth[:4]                // 연도 짤라내기. 글자수는 0부터 시작하고 끝 숫자는 빠진다. 0,1,2,3 위치 잘라내기
	curYearMonth = curYearBuf + curYearMonth[5:7] // 월 짤라내기

	st = "site" + curYearMonth // ""사이트"" 테이블명 확정
	//db.Exec("TRUNCATE TABLE "+st)

	/******************	설정값 변경 지시	********************/
	//디비에서 읽어와서 결정한다
	db, err := sql.Open("mysql", "root:1111@tcp("+dbLocation+":"+db_port+")/map")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	s := new(siteApTableVal)

	// update map.site201612 set flagFirm = "0", flagConf = "0" where macAddr = "00:06:7a:e7:20:88";
	if n.macAddr == "" {
		fmt.Println("\nAP 플래그 리셋(0)안됨.. MAC 주소를 쿼리로 보내주세요 :", s.flagConf, s.flagFirm, s.macAddr)
		return
	}
	querySt := "update " + st + " set flagFirm = \"0\", flagConf = \"0\" WHERE macAddr = " + "\"" + n.macAddr + "\""
	fmt.Println(querySt, "IP:", n.ipAddr) // IP까지 표시
	up, err := db.Prepare(querySt)
	if err != nil {
		panic(err)
	}
	defer up.Close()
	_, err = up.Exec()
	if err != nil {
		// the query, Scan returns ErrNoRows. 조건에 맞는 row가 없으면 프로그램이 종료되버림. ㅠ.ㅠ
		if err == sql.ErrNoRows {
			// there were no rows, but otherwise no error occurred
			fmt.Println("DB 에러 발생 -- [젬피]", sql.ErrNoRows)
		} else {
			fmt.Println("해당 MAC 주소의 row가 없습니다 !! -- [젬피]", sql.ErrNoRows)
			//log.Fatal(err)
		}
	}
	webSocString += "\n*** " + n.macAddr + " AP가 플래그를 리셋하였습니다. 테이블 = " + st + "\n"
	//fmt.Println("------------------------->>>",  s.flagConf, s.flagFirm, s.macAddr) // 플래그 값 확인용
}
/*
func shownode(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	askstart := r.Form.Get("show")
	askstart = "" + askstart
	//이제 여기다가 노드맥.txt파일 열어서 수정하면 된다. 마지막 ,는 삭제하던지 다른걸로 바꾸고 맨 앞에
	//"links": [ \n 정보정보정보 ,지우고 ]\n} 하면 될듯?
	if askstart == "1" {
		data, err := ioutil.ReadFile("nodeMac.txt")
		nodedata := string(data)
		if err != nil {
			fmt.Println("nodeMac파일이 아직 생성되지 않았습니다.")
		} else if nodedata != "" {
			a := len(nodedata)
			nodedata = nodedata[:a-2]
			nodedata = "{\"links\": [\n" + nodedata + "]\n}"
			//fmt.Println(nodedata)
			err = ioutil.WriteFile("./file_storage/gentelella-master/production/nodMac.json", []byte(nodedata), os.FileMode(644))
			ioutil.WriteFile("nodeMac.txt", []byte(""), os.FileMode(644))

			if err != nil {
				fmt.Println(err)
			}
		} else {
			fmt.Println("Data is NULL")
		}
	}
}
*/
func settmpLicense(w http.ResponseWriter, r *http.Request) {
	var b, bb, bbb string
	r.ParseForm()
	tmp := r.Form.Get("rore")
	mac := r.Form.Get("macAddr")
	tmp = "" + tmp
	if tmp == "write" {
		indextemp, err := ioutil.ReadFile("./file_storage/gentelella-master/production/index.html")
		if err != nil {
			panic(err)
		}
		index1temp, err := ioutil.ReadFile("./file_storage/gentelella-master/production/index1.html")
		if err != nil {
			panic(err)
		}
		index2temp, err := ioutil.ReadFile("./file_storage/gentelella-master/production/index2.html")
		if err != nil {
			panic(err)
		}
		a := string(indextemp)
		a1 := string(index1temp)
		a2 := string(index2temp)
		b = strings.Replace(a, "table", mac+";table", 1)
		bb = strings.Replace(a1, "table", mac+";table", 1)
		bbb = strings.Replace(a2, "table", mac+";table", 1)

		ioutil.WriteFile("./file_storage/gentelella-master/production/index.html", []byte(b), os.FileMode(644))
		ioutil.WriteFile("./file_storage/gentelella-master/production/index1.html", []byte(bb), os.FileMode(644))
		ioutil.WriteFile("./file_storage/gentelella-master/production/index2.html", []byte(bbb), os.FileMode(644))
	}
	fmt.Println(tmp)
	fmt.Println(mac)
	w.Write([]byte("\n\n" + mac + " 장비를 임시 추가하였습니다.\n\n\n" +
		"             서버 재가동시 사라집니다." +
		"\n\n\n[Web Site] www.jmpsys.com [전화] 02-415-0020"))
}
func setHttpRouterForAPandFile() {
	// 80포트로 AP와 양방향 통신 : AP로 부터 주기적으로 수신하는 페이지, 주기적으로 센터 지시사항 내려주는 것도 함께 이루어짐
	rr := mux.NewRouter()
	rr.HandleFunc("/", talkingWithAp).Methods("GET")      // AP로 부터 주기적 데이타 수신하고 펌업 지시 페이지
	rr.HandleFunc("/put", myPutHandler).Methods("PUT")    // 차후 대비... PUT 시험용
	rr.HandleFunc("/resetflag", resetFlag).Methods("GET") // 펌웨어, 설정파일 업데이트 플래그 지우기

	go func() { // AP 양방향 통신용
		log.Fatal(http.ListenAndServe(":"+web_ap_port, rr))
	}()
	http.HandleFunc("/websocket", webSoc) // 웹소켓
	http.HandleFunc("/showmesh",showmesh)
	//http.HandleFunc("/s", webSoc) // 웹소켓
	// http.Handle("/dashboard", http.StripPrefix("/dashboard", http.FileServer(http.Dir("./gentelella-master"))))
	http.Handle("/", http.FileServer(http.Dir("./file_storage"))) // 변수사용 시도해보자
	fmt.Printf("File Serving %s on HTTP port: %s\n", "root", web_file_port)
	go func() {
		log.Fatal(http.ListenAndServe(":"+web_file_port, nil))
	}()
}
func uMap(w http.ResponseWriter, r *http.Request) {
	table = strings.Replace(table, ";", ",", -1)
	a := strings.Index(table, "table")
	fmt.Println(a)
	r.ParseForm()
	fmt.Println(table[:a-1])
	loc := r.Form.Get("location")
	mac := r.Form.Get("macAddr")
	fmt.Println(mac, loc)
	ioutil.WriteFile("./file_storage/gentelella-master/production/mapdata.dat", []byte(loc), os.FileMode(644))
	w.Write([]byte("\n" + table[:a-1] + "\n" + mac + "\n" + loc))
	if r.Form.Get("delete") == "yes" {

	}
}
func indoor(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()
	logo:=r.Form.Get("logo")
	imgdat:=r.Form.Get("imgdata")
	imgname:=r.Form.Get("imgname")
	deling:=r.Form.Get("delimg")
	if deling!=""{
	//	fmt.Println(delimg,"aaaaaaaaaaaaaaaaaa")
		err:=os.Remove("./file_storage/gentelella-master/production/parkingimage/"+deling)
		fmt.Println(err)
	}else{
		//fmt.Println(imgdat)
		imgdat=strings.Replace(imgdat,"최건영","+",-1)
		//fmt.Println(imgdat)
		imgdatarr:=strings.Split(imgdat,",")

		imgnamearr:=strings.Split(imgname,",")
		for i, _ := range imgdatarr {
			//fmt.Println(imgdatarr[i])
			//	fmt.Println(imgnamearr[i])
			sDec,_:= base64.StdEncoding.DecodeString(imgdatarr[i])
			ioutil.WriteFile("./file_storage/gentelella-master/production/parkingimage/"+imgnamearr[i],[]byte(sDec),os.FileMode(644))
		}
		files, err := ioutil.ReadDir("./file_storage/gentelella-master/production/parkingimage/")
		if err != nil {
			log.Fatal(err)
		}
		var filenametag string
		var filedattag string
		for _, f := range files {
			if strings.Contains(f.Name(),".png"){
				filenametag+="<li onclick=\"load(this)\" onmousedown=\"edable(this)\"><a><strong>"+f.Name()+"</strong></a></li>"
				filedattag+="<img style=\"display:none\"class=\"thumb\" src=\"parkingimage/"+f.Name()+"\" title=\""+f.Name()+"\"id=\""+f.Name()+"1\"/>"
			//	fmt.Println(f.Name(),"///////")
			}

		}
		//img:=r.Form.Get("img")
		//img=strings.Replace(img,"최건영",";",-1)
		//ioutil.WriteFile("test.png", []byte(img), os.FileMode(644))
		//aqq:=strings.Split(img,",")
		//for i,_:=range aqq{
		//	imgsplit:=strings.Split(aqq[i],"base64,")
		//	fmt.Println(imgsplit[1])
		//}
		logodat,_:=readLines("./file_storage/gentelella-master/production/indoor.html")
		var changedinfo string
		for i, _ := range logodat {
			//fmt.Println(logodat[i])
			//fmt.Println(logodat[i])
			//fmt.Println(mapdat[i])
			if strings.Contains(logodat[i], "var exmakevalue") {
				logodat[i] = "var exmakevalue="+logo
			}
			if strings.Contains(logodat[i],"<!--리스트 여기다가넣기-->"){
				//fmt.Println(filenametag)
				logodat[i]="<!--리스트 여기다가넣기-->"+filenametag
			}
			if strings.Contains(logodat[i],"<!--이미지 여기다가 넣기-->"){
				//fmt.Println(filedattag)
				logodat[i]="<!--이미지 여기다가 넣기-->"+filedattag
			}
			//fmt.Println(logodat[i])
			changedinfo += logodat[i] + "\n"
		}
		ioutil.WriteFile("./file_storage/gentelella-master/production/indoor.html", []byte(changedinfo), os.FileMode(644))

	//	fmt.Println(logo)
		w.Write([]byte("\n설정이 적용되었습니다.\n"))
		//mg2html := "<html><body><img src=\"data:image/png;base64," + imgBase64Str + "\" /></body></html>"
		//w.Write([]byte(fmt.Sprintf(img2html)))

		//base 64를 파일로 저장하는 방법
	}
}
func changeprof(w http.ResponseWriter, r *http.Request){
	r.ParseForm()
	before:=r.Form.Get("before")
	if before=="yes"{

		w.Header().Set("Content-Type", "text/html")
		conf_table :=
			"<!DOCTYPE html> <html>	<html lang=\"en\"> <head><style>#wrapper{border: 2px solid blue;width:100%;}#wrapper2{border: 2px solid blue;width:100%;}</style>" + "<meta charset=\"UTF-8\"> </head> <body> <div id='wrapper'><div style='background-color:black;color:white'>JMP SYSTEMS Co., Ltd. 2017</div>"
		conf_table += "</div><div id='wrapper21' style='height:100px'><div style='float:left;height:100%;width:20%;'><input type='button'value='관리자에게 문의하세요.'></body> </html>"
		w.Write([]byte(conf_table))//현재 ID<br/>현재 PW</div><div style='float:right;height:100%;width:80%;'><input type='text'><br/><input type='password'></div></div>
	}else{

	}
}
func mactoweb(w http.ResponseWriter, r *http.Request){
	var ip string
	r.ParseForm()
	mactoip:=r.Form.Get("macnum")
	//fmt.Println(mactoip)
	db, err := sql.Open("mysql", "root:1111@tcp("+dbLocation+":"+db_port+")/map")
	//rw, err:=db.Query("SHOW TABLE STATUS FROM map")
	//fmt.Println("11111111111111",rw,err,"aaaaasdasdoqwojeliahsdlh")
	if err != nil {
		panic(err)
	}
		curYearMonth := time.Now().String()
		curYearBuf := curYearMonth[:4]                 // 연도 짤라내기. 글자수는 0부터 시작하고 끝 숫자는 빠진다. 0,1,2,3 위치 잘라내기
		curYearMonth = curYearBuf + curYearMonth[5:7]
		site := "site" + curYearMonth
		//n := new(siteApTableVal)
			mactoweb, err := db.Query("SELECT ipAddr FROM " + site + " where macAddr='" + mactoip + "'")
		for mactoweb.Next() {
			err := mactoweb.Scan(&ip)
			if err != nil {
			fmt.Println("\n사이트 테이블 읽어오는데서 에러 발생\n")
			//log.Fatal(err)
		}
	}
	defer db.Close()
	//fmt.Println(ip,"/////////////////")
	go open("http://"+ip)

}

func maploc(w http.ResponseWriter, r *http.Request) {
	maplocation, err := ioutil.ReadFile("./file_storage/gentelella-master/production/mapdata.dat")
	if err!=nil{
		ioutil.WriteFile("./file_storage/gentelella-master/production/mapdata.dat", []byte(defaultlodat), os.FileMode(644))
	}
	//fmt.Println(string(maplocation), err)
	maphtml, _ := readLines("./file_storage/gentelella-master/production/mapmak.html")
	var changedinfo string
	for i, _ := range maphtml {
		//fmt.Println(mapdat[i])
		if strings.Contains(maphtml[i], "var locations") {
			maphtml[i] = "var locations = " + string(maplocation)
		}
		changedinfo += maphtml[i] + "\n"
	}
	ioutil.WriteFile("./file_storage/gentelella-master/production/mapmak.html", []byte(changedinfo), os.FileMode(644))
}
func readLine(path string) {
	inFile, err := os.Open(path)
	if err != nil {
		fmt.Println(err.Error() + `: ` + path)
		return
	} else {
		defer inFile.Close()
	}
	scanner := bufio.NewScanner(inFile)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		fmt.Println(scanner.Text()) // the line
	}
}
func exchange(w http.ResponseWriter, r *http.Request){
	r.ParseForm()
	mac:=r.Form.Get("macaddr")
	mac = strings.Replace(mac, ":", "", -1)
	macTag := mac[len(mac)-6:]
	my7zip(macTag)
	w.Header().Set("Content-Type", "text/html")
	conf_table :=
		"<!DOCTYPE html> <html>	<html lang=\"en\"> <head> <style> table {width:100%;} table, th, td {" + " border: 1px solid black; border-collapse: collapse;	} th, td { padding: 5px; text-align: center;" + "} table.names tr:nth-child(even) { background-color: #eee; }	table.names tr:nth-child(odd) {" + "background-color:#fff; } table.names th { background-color: black; color: yellow } </style>" + "<meta charset=\"UTF-8\"> </head> <body> <p>JMP SYSTEMS Co., Ltd. 2017</p><p></p><p></p><p></p><table class=\"names\"> <tr> <th>Zip 저장완료 </th>" + "<th>MAC: " + mac + "</th> </tr>"
	conf_table += "</table><div><p></p>ctl+w 또는 닫기 클릭하세요 <button id='close'onclick=\"self.close()\"> 닫기 </button><p></p></div>"
	conf_table += "<p>메인보드 유선설정</p>"
	conf_table += "</body> </html><script>function LockF5() {if(event.ctrlKey&&event.keyCode==83){var button = document.getElementById('close');button.click();return false}console.log(event.keyCode);}document.onkeydown = LockF5;</script>"

	w.Write([]byte(conf_table))
	w.Write([]byte(mac + " 장비의 설정파일이 zip으로 저장되었습니다\r\n"))
}
func setHttpRouter() {
	r := mux.NewRouter()
	// 라우터는 경로와 핸들러 함수로 이루어짐
	// 8080 포트로 사용자 웹 서비스 : uManager 3.3 GUI 구현
	r.HandleFunc("/login", uMrRoot) // 로그인 화면
	r.HandleFunc("/", uMrRoot1)     // 로그인 화면
	r.HandleFunc("/maploc", maploc)
	//r.HandleFunc("/shownode", shownode).Methods("GET")
	r.HandleFunc("/indoor", indoor).Methods("GET")
	r.HandleFunc("/showmeshid", showmeshid).Methods("GET")
	r.HandleFunc("/savemesh", savemesh).Methods("GET")
	r.HandleFunc("/mactoweb", mactoweb).Methods("GET")
	r.HandleFunc("/savemeshconfig", savemeshconfig).Methods("GET")
	//r.HandleFunc("/delimg", delimg).Methods("GET")
	r.HandleFunc("/umap", uMap).Methods("GET")         // map 저장
	r.HandleFunc("/changeprof", changeprof).Methods("GET")
	r.HandleFunc("/aplist", showApList).Methods("GET") // 메인 화면 = AP 리스트
	r.HandleFunc("/reloadaplist", reloadApList).Methods("GET")
	r.HandleFunc("/truncate", truncate).Methods("GET")             // AP 리스트 재생성
	r.HandleFunc("/editconfig", editConfig).Methods("GET")         // 설정값 편집
	r.HandleFunc("/saveconfig", saveConfig).Methods("GET")         // 설정값 편집
	r.HandleFunc("/setmulticonfig", setMultiConfig).Methods("GET") // 설정값 편집
	r.HandleFunc("/syslog", showsyslog).Methods("GET")
	r.HandleFunc("/showdata", showData).Methods("GET")               // AP 테이블 표출
	r.HandleFunc("/maketable", makeTable).Methods("GET")             // 1번 테이블 만들기 (AP)
	r.HandleFunc("/event_maketable", event_makeTable).Methods("GET") // 2번 테이블 만들기 (uM)
	r.HandleFunc("/site_maketable", site_makeTable).Methods("GET")   // 3번 테이블 만들기 (Site)
	r.HandleFunc("/setflagall", setFlagAll).Methods("GET")           // 3번 테이블 만들기 (Site)
	r.HandleFunc("/setlicense", settmpLicense).Methods("GET")
	r.HandleFunc("/writing", writing).Methods("GET")
	r.HandleFunc("/loadconfig", loadconfig).Methods("GET")
	r.HandleFunc("/exchange",exchange).Methods("GET");
	go func() { // 사용자를 위한 GUI, 웹 서비스 용도
		err:=http.ListenAndServe(":"+web_user_port, r)
		fmt.Println(err)
		if err!=nil{
			open("http://"+db_IP+":8080/login") // 로그인 페이지 로딩...
					}

	}()
}

// ****************** DB 관련 함수 ***********************
// DB 관련 에러체크 소스코드 절약
func checkError(err error) {
	if err != nil {
		fmt.Println(err)
		fmt.Println("DB 관련 에러 발생--[젬피]")
		//panic(err)
	}
}

// readLines reads a whole file into memory and returns a slice of its lines.
func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// 웹서비스 및 DB 관련 url 사용여부, IP, port 설정파일 읽어서 실행 준비하기
func setConfig() {
	lines, err := readLines("config.txt")
	if err != nil {
		log.Fatalf("readLines: %s", err)
	}
	//fmt.Println("=====================", lines)
	i := 0
	for _, line := range lines {
		//	fmt.Println(i, line)
		s := strings.Split(line, "=")
		key, val := s[0], s[1]
		//	fmt.Println(key, val)
		switch key {
		case "[web_ap_port]":
			web_ap_port = val
		case "[web_file_port]":
			web_file_port = val
		case "[web_user_port]":
			web_user_port = val
		case "[db_use_url]":
			db_use_url = val
		case "[db_url]":
			db_url = val
		case "[db_IP]":
			db_IP = val
		case "[db_port]":
			db_port = val
		case "[url_of_Conf_75xx]":
			url_of_Conf_75xx = db_url+":"+web_file_port+"/"+val
		case "[url_of_Conf_77xx]":
			url_of_Conf_77xx= db_url+":"+web_file_port+"/"+val
		case "[url_of_Conf_922x]":
			url_of_Conf_922x = db_url+":"+web_file_port+"/"+val
		case "[url_of_Conf_50xx]":
			url_of_Conf_50xx = db_url+":"+web_file_port+"/"+val
		case "[url_of_Conf_20xx]":
			url_of_Conf_20xx = db_url+":"+web_file_port+"/"+val
		case "[url_of_Conf_920x]":
			url_of_Conf_920x = db_url+":"+web_file_port+"/"+val
		case "[url_of_Conf_50xx_mesh]":
			url_of_Conf_50xx_mesh = db_url+":"+web_file_port+"/"+val
		case "[url_of_Firm]":
			url_of_Firm = db_url+":"+web_file_port+"/"+val
		case "[debug_level]":
			debug_level = val
			//장비추가
		default:
			fmt.Println("\n\nuManager 시스템 설정 파일 오류\n")
		}
		i++
	}
	fmt.Printf("설정 파일 항목 %d 개를 읽었습니다\n", i)
}

// open opens the specified URL in the default browser of the user.
func open(url string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "windows":
		runos="windows"
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		runos="darwin"
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		runos="default"
		cmd = "xdg-open"
	}


	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

/*
func status(){
	// sql.DB 객체 생성
	db, err := sql.Open("mysql", "root:1111@tcp(127.0.0.1:3306)/map")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	// 복수 Row를 갖는 SQL 쿼리
	curYearMonth := time.Now().String()
	curYearBuf := curYearMonth[:4]                // 연도 짤라내기. 글자수는 0부터 시작하고 끝 숫자는 빠진다. 0,1,2,3 위치 잘라내기
	curYearMonth = curYearBuf + curYearMonth[5:7] // 월 짤라내기
	sFile,err :=os.Open("ap201704.ibd")
	if err != nil {
		log.Fatal(err)
	}
	defer sFile.Close()
	eFile, err:=os.Create("test_copy.ibd")
	if err != nil {
		log.Fatal(err)
	}
	defer eFile.Close()

	_, err=io.Copy(eFile,sFile)
	if err != nil {
		log.Fatal(err)
	}

	err = eFile.Sync()
	if err != nil {
		log.Fatal(err)
	}
}
*/
func setsyslog() {
	channel := make(syslog.LogPartsChannel) //channel 생성, syslog 메세지 저장하는 map 스트럭쳐해서 key 저장되있을듯
	handler := syslog.NewChannelHandler(channel)
	server := syslog.NewServer()
	server.SetFormat(syslog.Automatic)
	server.SetHandler(handler)
	server.ListenUDP(db_url + ":514")
	server.Boot()
	//f,_ = os.OpenFile("asdf.txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	go func(channel syslog.LogPartsChannel) {
		for logParts := range channel {
			timenow := time.Now().String()
			//fmt.Println(logParts,"//")
			s := strings.Split(logParts["client"].(string), ":")
			//fmt.Println(syslogmac[s[0]],s[0],"맥맥맥")
			strings1 := fmt.Sprint("[", timenow[0:16], "]", " log_Msg: ", logParts["tag"], ", ", logParts["content"], "\n")
			if len(s[0]) > 5 {
				f, err := os.OpenFile("./syslog/"+syslogmac[s[0]]+".txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
				if err != nil {
					panic(err)
				}
				if _, err = f.WriteString(strings1); err != nil {
					panic(err)
				}
			}

		}
	}(channel)
}

//st = "site" + curYearMonth                    // ""사이트"" 테이블명 확정.. fmt.Println("현재 선택된 테이블 : ",st) // 디버깅 중요정보
/****************** 신규발견시 자동 등록 ********************/
/*
db, err = sql.Open("mysql", "root:1111@tcp("+dbLocation+":"+db_port+")/map")
if err != nil {
panic(err)
}
defer db.Close()

s := new(siteApTableVal)
querySt := "SELECT macAddr from " + st + " WHERE macAddr = " + "\"" + n.macAddr + "\"" /// 테이블이 전혀 다르다!!! 하지만 물어볼땐 기존 n 구조체 MAC을 사용
//fmt.Println("===", querySt)   // 앞으로 모두 이렇게 코딩하자... 그래야 디버깅 된다. "보이게하라. 문제지역을 조명탄으로.."
err = db.QueryRow(querySt).Scan(&s.macAddr)
*/
func pingfunc(){
	//fmt.Println("여기여기")
	errcheck=make(map[string]string)
	pingresult=make(map[string]bool)
	for{
		//fmt.Println(errcheck)
		for mac,_ :=range errcheck{
			go pinginfo(mac)
			//fmt.Println(mac,ip)
		}

		time.Sleep(time.Millisecond*1000*2)
		//fmt.Println(pingresult)
		time.Sleep(time.Millisecond*1000*2)
	}
}
func pinginfo(mac string){
	//fmt.Println("192.168.0."+strconv.Itoa(a))
	mutex.Lock()
	pingresult[mac]=false
	mutex.Unlock()
	p := fastping.NewPinger()
	ra, err := net.ResolveIPAddr("ip4:icmp", errcheck[mac])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	p.AddIPAddr(ra)
	p.OnRecv = func(addr *net.IPAddr, rtt time.Duration) {
		//fmt.Printf("IP Addr: %s receive, RTT: %v\n", addr.String(), rtt)
		mutex.Lock()
		pingresult[mac]=true
		mutex.Unlock()
	}
	p.OnIdle = func() {
		//fmt.Println("finish")
	}
	err = p.Run()
	if err != nil {
		fmt.Println(err)
	}
	if pingresult[mac]==false{
		if errcheck[mac]==""{
			print(mac+" uM 최초 접속되지 않은 장비\n")
		}else{
			print(errcheck[mac]+"은 죽어있다."+mac+" \n")
		}

	}else{
		print(errcheck[mac]+"은 살아있다.\n")
	}
}
func main() {
	meshmapstring=make(map[string]string)
	hopcount = make(map[string]int)
	txmap = make(map[string]uint64)
	rxmap = make(map[string]uint64)
	tx = make(map[string]uint64)
	rx = make(map[string]uint64)
	rxtxmap = make(map[string]string)
	syslogmac = make(map[string]string) //syslog find mac using ip

	// 에러 이벤트명 설정
	// 시간은 기본적으로 보여주기
	/*
		eventName := map[int]string {
		1: "테이블 자료가 너무 많습니다",              // 라인 수 보여주기. 20만개 이상일때
		2: "AP가 주기적 보고를 하지 않습니다",         // MAC 주소 보여주기. 일단은 하나씩.. 일일이..
		3: "신규 AP가 보고를 시작하였습니다",          // MAC 주소 보여주기
		4: "최대 접속 단말 개수를 초과하였습니다",     // max 접속 초과 할때 해당 AP MAC 보여주기
		}
	*/

	cpuCoreCheckAndMaxProcs()    // consTimeLog 는 main 함수의 진행 흐름을 추적할 수 있도록 시간포함 디버깅 메시지 찍는다. 메인함수에서만 사용한다. 서브함수는 consTimeLogSubXXX 를 사용
	consTimeLog("최고 성능으로 동작합니다") // AP 및 사용자 웹통신 설정과 서비스 개시
	setConfig()                  // 설정파일 읽어서 글로벌 변수 설정
//	setTrafic()                  // 트래픽 읽어서 맵으로 저장

	dbLocation = db_url
	if db_use_url != "yes" { // yes 외의 문자일경우 IP 값을 사용한다
		dbLocation = db_IP
	}
	tt := "root:**********@tcp(" + dbLocation + ":" + db_port + ")/map"
	fmt.Println(tt)
	//db, err := sql.Open("mysql", "root:1111@tcp("+ dbLocation + ":" + db_port +")/map")
	db, err := sql.Open("mysql", "root:1111@tcp("+dbLocation+":"+db_port+")/map")
	//_, err = db.Exec("CREATE DATABASE map")
	//db.Query("SELECT * FROM `site201704` into outfile 'localhost:8100/aaa.svc' fields terminated by ','")
	defer db.Close() // 디비 너무 자주 빠르게 열고 닫지마세요
	if err != nil {
		fmt.Println("ERROR :: DB open 하지 못함 [젬피]")
		log.Fatal(err) // 에러 메시지 찍고 프로그램 종료
	} // Close the statement when we leave main() / the program terminates
	create("myidpw",4)
	if err != nil {
		panic(err)
	}
	err = db.Ping()
	if err != nil {
		fmt.Print("ERROR :: mySQL DB 접속 실패 또는 해당 스키마가 없습니다.\n")
		log.Fatal(err) // 에러 메시지 찍고 프로그램 종료
	}
	fmt.Println("DB ping 완료.. DB 위치 : ", tt)
	fmt.Println("DB 확인작업 시간 5초 입니다")

	// site DB row수 확인 - 장비 수 확인
	/*
		err = db.QueryRow("SELECT COUNT(*) FROM " + "site201701").Scan(&gApCnt)  // todo 01.22 날짜로 자동변화하도록... st등
		print( "site201701" + " 테이블의 Row : ",gApCnt , "\n") // todo 01.22 날짜로 자동변화하도록... st등
		webSocString += "-A05-" + fmt.Sprintf("%03d",gApCnt)
		print( webSocString, "\n")
	*/
	setsyslog() //syslog 수신시작
	time.Sleep(time.Millisecond * 500)
	setHttpRouter() // 관리자 명령 수신 시작
	time.Sleep(time.Millisecond * 500)
	consTimeLog("DB 점검중...")
	time.Sleep(time.Millisecond * 500)
	go setHttpRouterForAPandFile() // AP 수신 및 파일 서버 서비스 시작
	time.Sleep(time.Millisecond * 500)
	consTimeLog("AP 모니터링 서비스가 시작되었습니다.")

	querySt:="SELECT id, pw from myidpw WHERE admin = \"admin\""
	err = db.QueryRow(querySt).Scan(&userid,&userpw)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("새로운 MAC 발견하여 신규 장비등록 진행합니다...", sql.ErrNoRows)
			// 이제 진짜로 site DB에 넣는다....
			stmtIns, err := db.Prepare("INSERT INTO myidpw "+ myidpwinsert) // 테이블명 + 집어넣는 필드와 플레이스 홀더. INTO 뒤에 그리고 st 다음에도 공백하나 넣어라..
			if err != nil {

				fmt.Println("사이트 테이블 인써트 준비 하는데서 에러발생!!!!!--[시스템 점검 필요 !!]")
			}
			// n.id 대신 n.macAddr 로 변경한다.
			_, err = stmtIns.Exec("","","admin")
			checkError(err)
			// DB에서만 잘 관리하고 있으면 자동 정렬된 화면 출력기능으로 첵빡 테이블 파일 만들수 있으니까 당분간 ok
			// ROW 갯수 확인하는 방법
			// disk & cpu 부하줄이기 운동
		}else{
			fmt.Println(userid,userpw,"///////////////////")
		}
	}
	//fmt.Println(userid,userpw,"///////////////////")//ID랑 PW

	// todo : localhost 일경우 필요
	//go open("http://www.umanager.kr:8100/gentelella-master/production/index.html")
	//go open("http://localhost:8100/gentelella-master/production/") // 웹속 부르는 쪽도 똑같이 localhost 이어야 함. 127.0.0.1 은 오리진 에러 유발함
	go open("http://" + db_url + ":8080/login") // 웹속 부르는 쪽도 똑같이 localhost 이어야 함. 127.0.0.1 은 오리진 에러 유발함
	timenow := time.Now().String()
	ddata, err := ioutil.ReadFile("./log/" + timenow[0:7] + "-servertime.csv")
	if err != nil {
		ioutil.WriteFile("./log/" + timenow[0:7] + "-servertime.csv",[]byte(""),os.FileMode(644))
		fmt.Println(err)
	}
	servertime := string(ddata) + timenow[0:19] + " 서버 시작됨-----------------\n"
	err = ioutil.WriteFile("./log/"+timenow[0:7]+"-servertime.csv", []byte(servertime), os.FileMode(644))
	if err != nil {
		fmt.Println(err)
	}
	go pingfunc()
	////////////////////////////////////////////////////////////////////////////////////////////////////
	i := 0
	for i < 0 { // 10000000000000000
	//	fmt.Println(i, "asdasd")
		fmt.Printf("%016d.", i)
		if r := i % 320000000; r == 0 {
			fmt.Printf("\r\n")
		}
		i = i + 1000000 // 백만씩 더해보자
	}

	for spend_time < 2592000*2 { // 2달로 세팅함    // 8,640 x 5분 = 1개월
		if spend_time%10 == 0 && spend_time != 0 {


		}
		if q := spend_time % 60; q == 0 {
			fmt.Printf("%04d.", spend_time/60) //1분마다
			if (spend_time/60)%5 == 0 && spend_time != 0 {
				timenow = time.Now().String()
				ddata, _ = ioutil.ReadFile("./log/" + timenow[0:7] + "-servertime.csv")
				servertime = string(ddata) + timenow[0:19] + " 정상 동작중....\n"
				ioutil.WriteFile("./log/"+timenow[0:7]+"-servertime.csv", []byte(servertime), os.FileMode(644))
			}
			//err = db.QueryRow("SELECT COUNT(*) FROM " + dbrow).Scan(&gApCnt)
			//print(dbrow+"\n테이블의 Row : ", gApCnt, "\n테이블의 용량 : ",gApCnt*1300/(1024*1024),"MB","\n")
		}
		time.Sleep(time.Millisecond * 1 * 1000) // 1초
		if r := spend_time % 3600; r == 0 {
			fmt.Printf("\r\n")
			consTimeLog("동작중...") //1시간마다
		}
		spend_time++
	}
}

/******************* 사용법 **********************
1. 설정파일 확인
1.1. 데이타베이스 url 또는 IP가 제대로 되어있는지 확인
1.2. 펌웨어파일, 설정파일 파일명이 맞는지 확인

2. 다운로드 파일 형식 (날짜+a~z) 에 맞춰 웹서버 실행파일 하부 file_storage 밑에 파일 있는지 확인
************************************************/

/********************	쿼리 시험하기	************************
[[ 관리자 메뉴 ]]
AP 테이블 만들기
http://umanager.kr:8080/maketable?yearmonth=201612
이벤트 테이블 만들기
http://umanager.kr:8080/event_maketable?yearmonth=201612
사이트 목록 테이블 만들기
http://umanager.kr:8080/site_maketable?yearmonth=201701

수동으로 업데이트 플래그 셋하기 (모두 업글해라..)
www.umanager.kr:8080/setflagall?flagFirm=1&flagConf=1&macAddr=all&magic=415$$
수동으로 특정 MAC AP만 업데이트 플래그 선별하여 셋하기 (업글하고.. 부팅유발자..)
www.umanager.kr:8080/setflagall?flagFirm=1&flagConf=1&macAddr=00:06:7a:e7:20:88&magic=415$$

site 테이블 자료 보기
http://umanager.kr:8080/aplist?yearmonth=201701
ap 테이블 자료 보기
http://umanager.kr:8080/showdata?yearmonth=201612쿼

[[ AP 메뉴 ]]
AP 에서 업데이트 플래그 리셋하기
http://www.umanager.kr/resetflag?macAddr=00:06:7a:e7:20:88&magic=415$$

수동으로 쿼리 부여 --> 1 row 인써트 : 단말 접속 없음
http://192.168.1.11/?bootFlag=0&ssidW0=Nopass&ssidW1=MAP7500_5G&chanW0=6&chanW1=149&macAddr=00:06:7a:20:21:e3&ipAddr=192.168.1.142&btVer=1174470656&fwVer=4278255616&devTemp=40&curMem=37&pingTime=0&assoCtW0=0&assoCtW1=0&assoDevT=0&leaveDevT=0&rxByteW0=1092870&txByteW0=103243442&rxByteW1=0&txByteW1=909496&rxPktW0=6384&txPktW0=855552&rxPktW1=0&txPktW1=3052&magic=415$$
수동으로 쿼리 부여 --> 1 row 인써트 : 단말 접속 시간 발생
http://umanager.kr/?bootFlag=0&macAddr=00:06:7a:e7:20:88&ipAddr=14.71.128.227&btVer=mt76xx-bt.16.12.24&fwVer=m7700-fw.16.12.21&ssidW0=Airces_2G&ssidW1=Aircess_5G&chanW0=11&chanW1=149&rxByteW0=12237134&txByteW0=24000124450&rxByteW1=3598714030&txByteW1=4609003000&rxPktW0=15043&txPktW0=23400&rxPktW1=381920&txPktW1=482012&assoCtW0=2&assoCtW1=35&devTemp=42&curMem=64000000&pingTime=72&assoDevT=1470975817&leaveDevT=1470978888&magic=415$$
수동으로 쿼리 부여 --> 1 row 인써트 : 부팅 이벤트 발생
http://umanager.kr/?bootFlag=1&macAddr=00:06:7a:e7:20:88&ipAddr=14.71.128.227&btVer=mt76xx-bt.16.12.24&fwVer=m7700-fw.16.12.21&ssidW0=Airces_2G&ssidW1=Aircess_5G&chanW0=11&chanW1=149&rxByteW0=12237134&txByteW0=24000124450&rxByteW1=3598714030&txByteW1=4609003000&rxPktW0=15043&txPktW0=23400&rxPktW1=381920&txPktW1=482012&assoCtW0=2&assoCtW1=35&devTemp=42&curMem=64000000&pingTime=72&assoDevT=-&leaveDevT=-&magic=415$$
********************************************************/

//////////////////////////////////////////////////////////    연구노트    /////////////////////////////////////////////////////////

/**************	 맵 ******************************
fmt.Println(eventName)  // map 이라고 표시되면서 전부 다 쌍으로 표시된다
fmt.Println(eventName[1]) // 그 key 에 해당되는 스트링만 표시된다

for k, v :=range eventName {
fmt.Println(k,v)  // iterate 하면서 다 찍어낸다
}
*************************************************/
/*********************  SQL 사용법 by sp  ***********************
	// 1. 선택하기 (내용 받아오기)
	// SELECT <컬럼 리스트> FROM <테이블 이름> WHERE <검색조건>
	// Example: SELECT FirstName, LastName, OrderDate FROM Orders WHERE OrderDate > '10/10/2010'

	// 2. 추가하기
	// INSERT INTO <테이블 이름> (<컬럼 리스트>) VALUES (<값>)
	// Example: INSERT INTO Orders (FirstName, LastName, OrderDate) VALUES ('John', 'Smith', '10/10/2010')

	// 3. 수정하기
	// UPDATE <테이블 이름> SET <컬럼1> = <값>, <컬럼2> = <값>, … WHERE <검색조건>
	// Example: UPDATE Orders SET FirstName = 'John', LastName = 'Who' WHERE LastName='Wo'

	// 4. 지우기
	// DELETE FROM <테이블 이름> WHERE <검색조건>
	// Example: DELETE FROM Orders WHERE OrderDate < '10/10/2010'

	// 5. 소팅
	// SELECT <컬럼 리스트> FROM <테이블 이름> WHERE <검색조건> ORDER BY <컬럼 리스트>
	// Example: SELECT FirstName, LastName, OrderDate FROM Orders WHERE OrderDate > '10/10/2010' ORDER BY OrderDate

	// 6. 테이블 생성
	// CREATE TABLE <테이블 이름> ( Column1 DataType, Column2 DataType, Column3 DataType, …. )
	// CREATE TABLE Orders ( FirstName CHAR(100), LastName CHAR(100), OrderDate DATE, OrderValue Currency )
****************************************************************

/************************* mySQL 필수 설정 하기 All about mySQL on JMP S/W *************************************

URL로 DB 접근하기
1. 공유기 포트포워딩 설정. nslookup으로 확인
2. mySQL 접근 IP 허용 (공유기 IP : 192.168.0.254등)

	// mySQL 커멘트로 다음과 같이 설정해주어야 원격에서 접속할 수 있다. mySQL 기본은 로컬접속만 허용 한다 !!
	// GRANT ALL ON map.* TO root@'192.168.0.254' IDENTIFIED BY '1111';
	// 특정 테이블만 허용 하려면 map.TheTable 로 특정해주면 됨
	// 여기서는 공유기 IP 192.168.0.254만 허용하는 거고 % 를 쓰면 다 허용

//DB에 접속하기 위한 계정@비밀번호/localhost:포트넘버/DB 이름(schema)
db, err := sql.Open("mysql", "root:1111@tcp(www.umanager.kr:3306)/map")
if err != nil {
fmt.Println("ERROR :: DB open 하지 못함 [젬피]")
// Just for example purpose. You should use proper error handling instead of panic
log.Fatal(err) // 에러 메시지 찍고 프로그램 종료
}
// Close the statement when we leave main() / the program terminates
defer db.Close() // 디비 너무 자주 빠르게 열고 닫지마세요

err = db.Ping()
if err != nil {
fmt.Print("ERROR :: mySQL DB 접속 실패 또는 해당 스키마가 없습니다 [젬피]\n")
}
consTimeLog("DB ping 완료...")

Tip
1. 테이블 생성시 첫글자는 반드시 소문자 알파벳 일것 (숫자 불가)
2. 생성시 IF NOT EXISTS 필요 (나중에 table 없을때 런타임 에러 방지)
3. data field 정의는 const 로 선언하여 중심잡도록 한다
***********************************************************************/

/****************	full column 정보 출력하기  ******************
	showCol, err:= db.Prepare("select * from INFORMATION_SCHEMA.columns where table_schema='map' and table_name='skn20161224' order by ordinal_position;")
	checkError(err)
	resultShow, err := showCol.Exec()
	if err!= nil{
		fmt.Println("++++++++++++에러다......\n")
	}
	fmt.Println(resultShow, "<--->", *showCol)
**************************************************************/

/******************* 	mysql 기본 사용법	 *******************
하나의 Row만을 리턴할 경우 QueryRow() 메서드를,
복수개의 Row를 리턴할 경우 Query() 메서드를 사용
반복동작 준비 시킬때는 Prepare() 메소드를 사용

	err = db.QueryRow("SELECT * FROM skn20161224 WHERE w1_tx > 13").Scan(&apPosi,&ssid,&apdate,&w1Tx)
	if err != nil {
		// Scan uses the first row and discards the rest. If no row matches
		// the query, Scan returns ErrNoRows. 조건에 맞는 row가 없으면 프로그램이 종료되버림. ㅠ.ㅠ
		if err == sql.ErrNoRows {
			// there were no rows, but otherwise no error occurred
			fmt.Println("QueryRow 에러 발생--[젬피]",sql.ErrNoRows)
		} else {
			log.Fatal(err)
		}
	}
	fmt.Println(apPosi, ssid, apdate, w1Tx)


		Opening and closing databases can cause exhaustion of resources.
		Failing to use rows.Close() can cause exhaustion of resources.
		Using Query() for a statement that doesn't return rows is a bad idea.
		Failing to use prepared statements can lead to a lot of extra database activity.
		Nulls cause annoying problems, which may show up only in production.

 The Query() will return a sql.Rows, which will not be released until it's garbage collected, which can be a long time.
 During that time, it will continue to hold open the underlying connection
****************************************************************/

/***********	Row 카운팅하기 ***********************
	var rowCnt int
	// QueryRow는 거대 구조체에 담겨오지 않고 간단히 err 만 리턴 받고 검색된 값은 Scan 포인터에 연결된다.
	// 테이블 존재 유무 조사.  없으면 이렇게 에러 뜬다 :: Error 1146: Table 'map.skn20161s224' doesn't exist
	err = db.QueryRow("SELECT COUNT(*) FROM skn20161224 WHERE w1_tx > ?", 0).Scan(&rowCnt)
	fmt.Println("8888",err)
	fmt.Println("현재 테이블에 있는 Row 수는",rowCnt)

	// Exec 메서드의 첫번째 파라미터에는 SQL문을 적고, 그 SQL문 안에 ? 이 있는 경우 계속해서 상응하는 파라미터를 넣어 준다.
	// Exec 메서드는 sql.Result와 error 객체를 리턴하며, sql.Result 객체로부터 갱신된 레코드수(RowsAffected())와 새로 추가된 Id (LastInsertId())를 구할 수 있다. <--???

	// Prepared Statement는 데이타베이스 서버에 Placeholder를 가진 SQL문을 미리 준비시키는 것으로,
	// 차후 해당 Statement를 호출할 때 준비된 SQL문을 빠르게 실행하도록 하는 기법

	// sql.DB의 Prepare() 메서드를 써서 Placeholder를 가진 SQL문을 미리 준비시키고, sql.Stmt 객체를 리턴받는다.
	// 차후 이 sql.Stmt 객체의 Exec (혹은 Query/QueryRow) 메서드를 사용하여 준비된 SQL문을 실행
**************************************************/

/***********	Placeholder *************************
	//SQL 쿼리에서 ? (Placeholder)를 사용하여 Parameterized Query를 사용
	// ? 조건에 맞는 것이 없어도 에러 발생하지 않는다. 그냥 Next()가 nil 이어서 루프진입이 안될뿐...
	rows, err := db.Query("select 설치위치, SSID, 설치일자, w1_tx from skn20161224 WHERE w1_tx > ?", 1)
	if err != nil {
		fmt.Println("db.Query 에서 에러발생--[JMP]")
		log.Fatal(err)
	}
	// 항상 에러 처리 바로 뒤에 위치한다. 루프 밖에 두면 에러가 날까? 루프내에 메모리누수 염려있다??
	defer rows.Close()
	rows.Close() is a harmless no-op if it’s already closed,
	so you can call it multiple times. Notice, however, that we check the error first,
	and only call rows.Close() if there isn’t an error, in order to avoid a runtime panic.
	You should always defer rows.Close(),
	even if you also call rows.Close() explicitly at the end of the loop, which isn’t a bad idea.

	// 리턴받은 구조체에서 실제 검색등은 메모리내 구조체를 가지고 수행한다
	// 메모리 내 rows 구조체에 포착된 내용들 출력해보자
	for rows.Next() {
		err := rows.Scan(&apPosi,&ssid,&apdate,&w1Tx)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(apPosi, ssid, apdate, w1Tx)
		fmt.Println(json.Marshal(rows))
		fmt.Println(rows)
	}
**************************************************/

/**************	 테이블 싹 지우기	************************
	delRow, err:= db.Prepare("DELETE from skn20161224")
	if err != nil {
		fmt.Println("db 지우는거 준비하는데서 에러발생--[JMP]")
		log.Fatal(err)
	}
	defer delRow.Close()

	result, err := delRow.Exec()
	if err != nil {
		fmt.Println("db 지우는데서 에러발생--[JMP]")
		log.Fatal(err)
	}
	affect, err := result.RowsAffected()
	checkError(err)
	fmt.Println("영향받은 Row 수는",affect,"입니다")
	fmt.Println("모든 필드 삭제 완료................")
************************************************************/

/*****************	 Exec 메서드 	*******************
	//Query/QueryRow 메서드는 데이타를 리턴할 때 사용하는 반면, DML과 같이 리턴되는 데이타가 없는 경우는 Exec 메서드를 사용
	//Data Manipulation Language
	stmtIns, err := db.Prepare("INSERT INTO skn20161224 (SSID,설치위치,설치일자,w1_tx) VALUES (?,?,?,?)")
	if err != nil {
		fmt.Println("db 인써트 준비 하는데서 에러발생!!!!!--[JMP]")
		log.Fatal(err)
	}
	defer stmtIns.Close()

	_, err = stmtIns.Exec("MAP7500_2G","천안-강릉","2016-09-09", 0 )
	checkError(err)


	for i:=0; i< 10; i++ {
		//fmt.Println("인써트 루프 횟수 :", i)
		_, err = stmtIns.Exec("강릉","MAP7500_2G", "2016-09-09", i*113 )
		checkError(err)
	}

*********************************************************/
/*************************	실행 시간 측정하기	********************
	startTime := time.Now()
	{
	시간 걸리는 루틴들....
	}
	// 경과 시간
	elapsedTime := time.Since(startTime)
	fmt.Printf("\n 실행시간: %s\n\n\n", elapsedTime)
***********************************************************************/

/******************** i 의 크기를 알기 위해 	********************
경까지 지켜봄.. 36년전에 비해 너무 크고 빠른 i

	i=0
	for i < 0 {  // 10000000000000000
		fmt.Printf("%016d.",i)
		if r := i%320000000; r ==0 {
			fmt.Printf("\r\n")
		}
		i = i+1000000 // 백만씩 더해보자
	}

	// 중간 보고 시키는 방법
	i=0
	for i < 60 {                            // 1시간
		fmt.Printf("%04d.",i)
		time.Sleep(time.Millisecond*1000*60) // 1분
		if r := i%60; r ==0 {
			fmt.Printf("\r\n")
			consTimeLog("동작중...")
		}
		i++
	}
**************************************************************/

/*************	특정일에 동작시키는 방법  **********************
	_, month, day := time.Now().Date()
	fmt.Println(month)
	if month == time.November && day == 10 {
		fmt.Println("Happy Go day!")
	}
************************************************************/
