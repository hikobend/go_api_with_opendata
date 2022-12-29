package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

type AutoGenerated struct {
	ErrorInfo struct {
		ErrorFlag    string `json:"errorFlag"`
		ErrorCode    string `json:"errorCode"`
		ErrorMessage string `json:"errorMessage"`
	} `json:"errorInfo"`
	ItemList []struct {
		Date      string `json:"date"`
		NameJp    string `json:"name_jp"`
		Npatients string `json:"npatients"`
	} `json:"itemList"`
}

type infection struct {
	Date      time.Time `json:"date"`
	NameJp    string    `json:"name_jp"`
	Npatients int       `json:"npatients"`
}

func main() {
	r := gin.Default()

	r.POST("/import", Import)
	r.GET("/gets", Get)
	r.GET("/get/:date", GetInfectionByDate)                                       // 日付を選択し、感染者を取得 47都道府県
	r.GET("/getInfection/:date1/:date2", GetBetweenDateNpatients)                 // 期間を選択し、感染者を取得 47都道府県
	r.GET("/npatients/:place/:date", GetDateNpatients)                            // 日付と地域を選択し、感染者を取得
	r.GET("/getnpatients/:place/:date1/:date2", GetBetWeenDateNpatientsWithPlace) // 期間を選択し、感染者を取得

	r.Run()
}

func GetBetWeenDateNpatientsWithPlace(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	place := c.Param("place")
	date1 := c.Param("date1")
	date2 := c.Param("date2")

	rows, err := db.Query("select date, name_jp, npatients from infection where name_jp = ? and date between ? and ?;", place, date1, date2)
	if err != nil {
		log.Fatal(err)
	}
	var resultInfection []infection

	for rows.Next() {
		infection := infection{}
		if err := rows.Scan(&infection.Date, &infection.NameJp, &infection.Npatients); err != nil {
			log.Fatal(err)
		}
		resultInfection = append(resultInfection, infection)
	}

	c.JSON(http.StatusOK, resultInfection)

}

func GetBetweenDateNpatients(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	date1 := c.Param("date1")
	date2 := c.Param("date2")

	rows, err := db.Query("select date, name_jp, npatients from infection where date between ? and ?", date1, date2)
	if err != nil {
		log.Fatal(err)
	}
	var resultInfection []infection

	for rows.Next() {
		infection := infection{}
		if err := rows.Scan(&infection.Date, &infection.NameJp, &infection.Npatients); err != nil {
			log.Fatal(err)
		}
		resultInfection = append(resultInfection, infection)
	}

	c.JSON(http.StatusOK, resultInfection)

}

func GetInfectionByDate(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	date := c.Param("date")

	rows, err := db.Query("select date, name_jp, npatients from infection where date = ?", date)
	if err != nil {
		log.Fatal(err)
	}
	var resultInfection []infection

	for rows.Next() {
		infection := infection{}
		if err := rows.Scan(&infection.Date, &infection.NameJp, &infection.Npatients); err != nil {
			log.Fatal(err)
		}
		resultInfection = append(resultInfection, infection)
	}

	c.JSON(http.StatusOK, resultInfection)
}

func Get(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query("select date, name_jp, npatients from infection where date between '2022-12-16T00:00:00Z' and '2022-12-17T00:00:00Z'")
	if err != nil {
		log.Fatal(err)
	}
	var resultInfection []infection

	for rows.Next() {
		infection := infection{}
		if err := rows.Scan(&infection.Date, &infection.NameJp, &infection.Npatients); err != nil {
			log.Fatal(err)
		}
		resultInfection = append(resultInfection, infection)
	}

	c.JSON(http.StatusOK, resultInfection)
}

func GetDateNpatients(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	date := c.Param("date")
	place := c.Param("place") // place用の別テーブルを作成して、そこのidを選択できないか。プルダウンで選択したい。

	var infection infection

	err = db.QueryRow("SELECT date, name_jp, npatients FROM infection WHERE name_jp = ? and date = ?", place, date).Scan(&infection.Date, &infection.NameJp, &infection.Npatients)

	if err != nil {
		log.Fatal(err)
	}
	c.JSON(http.StatusOK, infection)

}

func Import(c *gin.Context) { // データ取得、データベースに保存
	log.Print("データ取り込み中")
	url := "https://opendata.corona.go.jp/api/Covid19JapanAll"
	resp, _ := http.Get(url)
	if resp != nil {
		defer resp.Body.Close()
	}
	byteArray, _ := ioutil.ReadAll(resp.Body)

	jsonBytes := ([]byte)(byteArray)
	data := new(AutoGenerated)

	if err := json.Unmarshal(jsonBytes, data); err != nil {
		fmt.Println("JSON Unmarshal error:", err)
		return
	}
	// fmt.Println(data.ItemList)

	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	delete, err := db.Prepare("DELETE FROM infection")
	if err != nil {
		log.Fatal(err)
	}
	delete.Exec()

	for _, v := range data.ItemList {
		insert, err := db.Prepare("INSERT INTO infection(date, name_jp, npatients) values (?,?,?)")
		if err != nil {
			log.Fatal(err)
		}
		insert.Exec(v.Date, v.NameJp, v.Npatients)
	}

	log.Print("データ取り込み完了")
}
