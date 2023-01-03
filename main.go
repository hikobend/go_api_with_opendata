package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/go-playground/validator.v9"
)

type Npatients struct {
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

type Medical struct {
	FacilityId   string `json:"facilityId"`
	FacilityName string `json:"facilityName"`
	ZipCode      string `json:"zipCode"`
	PrefName     string `json:"prefName"`
	FacilityAddr string `json:"facilityAddr"`
	FacilityTel  string `json:"facilityTel"`
	Latitude     string `json:"latitude"`
	Longitude    string `json:"longitude"`
	SubmitDate   string `json:"submitDate"`
	FacilityType string `json:"facilityType"`
	AnsType      string `json:"ansType"`
	LocalGovCode string `json:"localGovCode"`
	CityName     string `json:"cityName"`
	FacilityCode string `json:"facilityCode"`
}

type infection struct {
	Date      time.Time `json:"date"`
	NameJp    string    `json:"name_jp"`
	Npatients int       `json:"npatients"`
}

type diff_Npatients struct {
	Npatients int `json:"npatients"`
}

type diff_Npatients_Place struct {
	NameJp        string `json:"name_jp"`
	Npatients     int    `json:"npatients"`
	NpatientsPrev int    `json:"npatientsprev"`
	Message       string `json:"message"`
}

type diff_Npatients_Place_Per struct {
	NameJp        string  `json:"name_jp"`
	Npatients     float64 `json:"npatients"`
	NpatientsPrev float64 `json:"npatientsprev"`
	Per           string  `json:"per"`
	Message       string  `json:"message"`
}

type Event_JSON struct {
	Title       string `json:"title" validate:"required"`
	Description string `json:"description"`
	Begin       string `json:"begin" validate:"required"`
	End         string `json:"end" validate:"required"`
}

type Medicals struct {
	FacilityName string `json:"facilityName"` // 病院名
	FacilityAddr string `json:"facilityAddr"` // 場所
	FacilityType string `json:"facilityType"` // 状況
}

type Medicals_show struct {
	FacilityName string `json:"facilityName"` // 病院名
	ZipCode      string `json:"zipCode"`      // 郵便番号
	PrefName     string `json:"prefName"`     // 都道府県
	FacilityAddr string `json:"facilityAddr"` // 場所
	FacilityTel  string `json:"facilityTel"`  // 電話番号
	SubmitDate   string `json:"submitDate"`   // 日付
	FacilityType string `json:"facilityType"` // 状況
	CityName     string `json:"cityName"`     // 市町村
}

func main() {
	r := gin.New()
	r.Use(loggingMiddleware())
	// ----------------------------------
	// デフォルトで表示
	// ----------------------------------
	r.GET("/count/:date", CountOfPatients)                     // 日の感染者の合計
	r.GET("/averagenpatients/:date", AverageNpatients)         // 日付を入力して、全国の感染者を上回った都道府県を表示
	r.GET("/averagenpatientsover/:date", AverageNpatientsOver) // 日付を入力して、全国の感染者を下回った都道府県を表示
	// ----------------------------------
	// 1
	// ----------------------------------
	r.GET("/firstfirst/:date", FirstFirst)   // 都道府県のマップを表示 色で危険地帯を視覚で把握可能 前々日比と前日比を算出して、前日比の方が多い場合、警告文字を変更する。その文字によって色を変える
	r.GET("/firstsecond/:date", FirstSecond) // 都道府県のマップを表示 色で危険地帯を視覚で把握可能 前々日比と前日比を算出して、前日比の方が多い場合、警告文字を変更する。その文字によって色を変える
	// ----------------------------------
	// 2
	// ----------------------------------
	r.GET("/secondfirst/:place/:date", SecondFirst)       // ここ7日間の感染者推移
	r.GET("/diffadd/:place/:date", DiffAdd)               // 前日比を表示
	r.GET("/npatientsinmonth/:place/:date", SecondSecond) // 年月と都道府県を取得して、その月の感染者数推移を取得
	r.GET("/npatientsinyear/:place/:date", SecondThird)   // 年と都道府県を取得して、その年の感染者推移を取得
	// ----------------------------------
	// 3
	// ----------------------------------
	r.POST("/create", Create)                               // コロナに関するメモを追加
	r.GET("/show/:id", Show)                                // コロナに関するメモを表示
	r.GET("/shows", ShowAll)                                // コロナに関するメモを表示
	r.PATCH("/show/:id", Update)                            // コロナに関するメモを変更
	r.DELETE("/delete/:id", Delete)                         // コロナに関するメモを削除
	r.GET("/getInfection/:date1/:date2", ThirdSecond)       // 期間を選択し、感染者を取得 47都道府県
	r.GET("/getnpatients/:place/:date1/:date2", ThirdThird) // 期間を選択し、感染者を取得
	// ----------------------------------
	// 4
	// ----------------------------------
	r.GET("/medical/:place", ForthFirst) // 都道府県のマップを表示 色で危険地帯を視覚で把握可能 前々日比と前日比を算出して、前日比の方が多い場合、警告文字を変更する。その文字によって色を変える

	// ----------------------------------
	// 5
	// ----------------------------------
	// ----------------------------------
	// データをimport
	// ----------------------------------
	r.POST("/import", Import)               // 都道府県感染者オープンAPIをimport
	r.POST("/importmedical", ImportMedical) // 都道府県感染者オープンAPIをimport

	r.Run()
}

func loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// ミドルウェアの前処理
		startTime := time.Now()

		// ミドルウェア内でリクエストの処理を実行
		c.Next()

		// ミドルウェアの後処理
		latency := time.Since(startTime)
		fmt.Println(c.Request.Method, c.Request.URL.Path, c.Writer.Status())
		time := fmt.Sprintf("%dms", latency/time.Millisecond)
		log.Print(time)
	}
}

func CountOfPatients(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer db.Close()

	date := c.Param("date")

	var sum int
	err = db.QueryRow("select sum(npatients) from infection where date = ?", date).Scan(&sum)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 結果をJSONで出力
	c.JSON(http.StatusOK, gin.H{
		"date":      date,
		"npatients": sum,
	})
}

func AverageNpatients(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	date := c.Param("date")

	rows, err := db.Query("select date, name_jp, npatients from infection where date = ? and npatients > (select avg(npatients) from infection)", date)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

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

func AverageNpatientsOver(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	date := c.Param("date")

	rows, err := db.Query("select date, name_jp, npatients from infection where date = ? and npatients < (select avg(npatients) from infection)", date)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

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

// -------------
// 1 - 1
// -------------

func FirstFirst(c *gin.Context) {

	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer db.Close()

	date, err := time.Parse("2006-01-02", c.Param("date"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	prevDate := date.AddDate(0, 0, -1)
	prev2Date := date.AddDate(0, 0, -2)

	infections := []diff_Npatients_Place{}
	var wg sync.WaitGroup

	places := []string{"北海道", "青森県", "岩手県", "宮城県", "秋田県", "山形県", "福島県", "茨城県", "栃木県", "群馬県", "埼玉県", "千葉県", "東京都", "神奈川県", "新潟県", "富山県", "石川県", "福井県", "山梨県", "長野県", "岐阜県", "静岡県", "愛知県", "三重県", "滋賀県", "京都府", "大阪府", "兵庫県", "奈良県", "和歌山県", "鳥取県", "島根県", "岡山県", "広島県", "山口県", "徳島県", "香川県", "愛媛県", "高知県", "福岡県", "佐賀県", "長崎県", "熊本県", "大分県", "宮崎県", "鹿児島県", "沖縄県"}
	for _, place := range places {
		wg.Add(1)
		go func(place string) {
			defer wg.Done()

			npatients := diff_Npatients_Place{NameJp: place}

			err = db.QueryRow("SELECT (SELECT npatients FROM infection WHERE date = ? AND name_jp = ?) - (SELECT npatients FROM infection WHERE date = ? AND name_jp = ?) as npatients", date, place, prevDate, place).Scan(&npatients.Npatients)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			err = db.QueryRow("SELECT (SELECT npatients FROM infection WHERE date = ? AND name_jp = ?) - (SELECT npatients FROM infection WHERE date = ? AND name_jp = ?) as npatients", prevDate, place, prev2Date, place).Scan(&npatients.NpatientsPrev)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			if npatients.Npatients/npatients.NpatientsPrev*100 > 140 {
				npatients.Message = "Too Danger"
			} else if npatients.Npatients/npatients.NpatientsPrev*100 > 120 {
				npatients.Message = "Danger"
			} else if npatients.Npatients/npatients.NpatientsPrev*100 > 100 {
				npatients.Message = "Warning"
			} else if npatients.Npatients/npatients.NpatientsPrev*100 > 80 {
				npatients.Message = "Caution"
			} else {
				npatients.Message = "attention"
			}
			infections = append(infections, npatients)
		}(place)
	}
	wg.Wait()

	c.JSON(http.StatusOK, infections)
}

// -------------
// 1 - 2
// -------------

func FirstSecond(c *gin.Context) {

	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer db.Close()

	date, err := time.Parse("2006-01-02", c.Param("date"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	prevDate := date.AddDate(0, 0, -1)
	prev2Date := date.AddDate(0, 0, -2)

	infections := []diff_Npatients_Place_Per{}
	var wg sync.WaitGroup

	places := []string{"北海道", "青森県", "岩手県", "宮城県", "秋田県", "山形県", "福島県", "茨城県", "栃木県", "群馬県", "埼玉県", "千葉県", "東京都", "神奈川県", "新潟県", "富山県", "石川県", "福井県", "山梨県", "長野県", "岐阜県", "静岡県", "愛知県", "三重県", "滋賀県", "京都府", "大阪府", "兵庫県", "奈良県", "和歌山県", "鳥取県", "島根県", "岡山県", "広島県", "山口県", "徳島県", "香川県", "愛媛県", "高知県", "福岡県", "佐賀県", "長崎県", "熊本県", "大分県", "宮崎県", "鹿児島県", "沖縄県"}
	for _, place := range places {
		wg.Add(1)
		go func(place string) {
			defer wg.Done()

			npatients := diff_Npatients_Place_Per{NameJp: place}

			err = db.QueryRow("SELECT (SELECT npatients FROM infection WHERE date = ? AND name_jp = ?) - (SELECT npatients FROM infection WHERE date = ? AND name_jp = ?) as npatients", date, place, prevDate, place).Scan(&npatients.Npatients)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			err = db.QueryRow("SELECT (SELECT npatients FROM infection WHERE date = ? AND name_jp = ?) - (SELECT npatients FROM infection WHERE date = ? AND name_jp = ?) as npatients", prevDate, place, prev2Date, place).Scan(&npatients.NpatientsPrev)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			per := npatients.Npatients / npatients.NpatientsPrev * 100
			s := strconv.Itoa(int(per))
			npatients.Per = s + "%"

			if npatients.Npatients/npatients.NpatientsPrev*100 > 140 {
				npatients.Message = "Too Danger"
			} else if npatients.Npatients/npatients.NpatientsPrev*100 > 120 {
				npatients.Message = "Danger"
			} else if npatients.Npatients/npatients.NpatientsPrev*100 > 100 {
				npatients.Message = "Warning"
			} else if npatients.Npatients/npatients.NpatientsPrev*100 > 80 {
				npatients.Message = "Caution"
			} else {
				npatients.Message = "attention"
			}
			infections = append(infections, npatients)
		}(place)
	}

	wg.Wait()

	c.JSON(http.StatusOK, infections)
}

// -------------
// 2 - 1
// -------------

func SecondFirst(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer db.Close()

	date, err := time.Parse("2006-01-02", c.Param("date"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}) // 400
		return
	}
	prevDate := date.AddDate(0, 0, -1)
	prev2Date := date.AddDate(0, 0, -2)
	prev3Date := date.AddDate(0, 0, -3)
	prev4Date := date.AddDate(0, 0, -4)
	prev5Date := date.AddDate(0, 0, -5)
	prev6Date := date.AddDate(0, 0, -6)

	place := c.Param("place")

	var infection1 infection
	var infection2 infection
	var infection3 infection
	var infection4 infection
	var infection5 infection
	var infection6 infection
	var infection7 infection
	var wg sync.WaitGroup
	go func() {
		defer wg.Done()
		err = db.QueryRow("SELECT date, name_jp, npatients FROM infection WHERE name_jp = ? and date = ?", place, date).Scan(&infection1.Date, &infection1.NameJp, &infection1.Npatients)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
			return
		}
	}()
	go func() {
		defer wg.Done()
		err = db.QueryRow("SELECT date, name_jp, npatients FROM infection WHERE name_jp = ? and date = ?", place, prevDate).Scan(&infection2.Date, &infection2.NameJp, &infection2.Npatients)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
			return
		}
	}()
	go func() {
		defer wg.Done()
		err = db.QueryRow("SELECT date, name_jp, npatients FROM infection WHERE name_jp = ? and date = ?", place, prev2Date).Scan(&infection3.Date, &infection3.NameJp, &infection3.Npatients)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
			return
		}
	}()
	go func() {
		defer wg.Done()
		err = db.QueryRow("SELECT date, name_jp, npatients FROM infection WHERE name_jp = ? and date = ?", place, prev3Date).Scan(&infection4.Date, &infection4.NameJp, &infection4.Npatients)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
			return
		}
	}()
	go func() {
		defer wg.Done()
		err = db.QueryRow("SELECT date, name_jp, npatients FROM infection WHERE name_jp = ? and date = ?", place, prev4Date).Scan(&infection5.Date, &infection5.NameJp, &infection5.Npatients)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
			return
		}
	}()
	go func() {
		defer wg.Done()
		err = db.QueryRow("SELECT date, name_jp, npatients FROM infection WHERE name_jp = ? and date = ?", place, prev5Date).Scan(&infection6.Date, &infection6.NameJp, &infection6.Npatients)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
			return
		}
	}()
	go func() {
		defer wg.Done()
		err = db.QueryRow("SELECT date, name_jp, npatients FROM infection WHERE name_jp = ? and date = ?", place, prev6Date).Scan(&infection7.Date, &infection7.NameJp, &infection7.Npatients)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
			return
		}
	}()
	wg.Add(7)
	wg.Wait()

	infections := []infection{infection1, infection2, infection3, infection4, infection5, infection6, infection7}

	c.JSON(http.StatusOK, infections)
}

func DiffAdd(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer db.Close()

	place := c.Param("place")
	date, err := time.Parse("2006-01-02", c.Param("date"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}) // 400
		return
	}

	prevDate := date.AddDate(0, 0, -1)
	prev2Date := date.AddDate(0, 0, -2)
	prev3Date := date.AddDate(0, 0, -3)
	prev4Date := date.AddDate(0, 0, -4)
	prev5Date := date.AddDate(0, 0, -5)
	prev6Date := date.AddDate(0, 0, -6)

	var diff1 diff_Npatients
	var diff2 diff_Npatients
	var diff3 diff_Npatients
	var diff4 diff_Npatients
	var diff5 diff_Npatients
	var diff6 diff_Npatients

	var wg sync.WaitGroup
	wg.Add(6)
	go func() {
		err = db.QueryRow("SELECT (SELECT npatients FROM infection WHERE date = ? AND name_jp = ?) - (SELECT npatients FROM infection WHERE date = ? AND name_jp = ?) as npatients", date, place, prevDate, place).Scan(&diff1.Npatients)
		if err != nil {
			log.Fatal(err)
		}
		wg.Done()
	}()
	go func() {
		err = db.QueryRow("SELECT (SELECT npatients FROM infection WHERE date = ? AND name_jp = ?) - (SELECT npatients FROM infection WHERE date = ? AND name_jp = ?) as npatients", prevDate, place, prev2Date, place).Scan(&diff2.Npatients)
		if err != nil {
			log.Fatal(err)
		}
		wg.Done()
	}()
	go func() {
		err = db.QueryRow("SELECT (SELECT npatients FROM infection WHERE date = ? AND name_jp = ?) - (SELECT npatients FROM infection WHERE date = ? AND name_jp = ?) as npatients", prev2Date, place, prev3Date, place).Scan(&diff3.Npatients)
		if err != nil {
			log.Fatal(err)
		}
		wg.Done()
	}()
	go func() {
		err = db.QueryRow("SELECT (SELECT npatients FROM infection WHERE date = ? AND name_jp = ?) - (SELECT npatients FROM infection WHERE date = ? AND name_jp = ?) as npatients", prev3Date, place, prev4Date, place).Scan(&diff4.Npatients)
		if err != nil {
			log.Fatal(err)
		}
		wg.Done()
	}()
	go func() {
		err = db.QueryRow("SELECT (SELECT npatients FROM infection WHERE date = ? AND name_jp = ?) - (SELECT npatients FROM infection WHERE date = ? AND name_jp = ?) as npatients", prev4Date, place, prev5Date, place).Scan(&diff5.Npatients)
		if err != nil {
			log.Fatal(err)
		}
		wg.Done()
	}()
	go func() {
		err = db.QueryRow("SELECT (SELECT npatients FROM infection WHERE date = ? AND name_jp = ?) - (SELECT npatients FROM infection WHERE date = ? AND name_jp = ?) as npatients", prev5Date, place, prev6Date, place).Scan(&diff6.Npatients)
		if err != nil {
			log.Fatal(err)
		}
		wg.Done()
	}()
	wg.Wait()

	infections := []diff_Npatients{diff1, diff2, diff3, diff4, diff5, diff6}

	c.JSON(http.StatusOK, infections)
}

// -------------
// 2 - 2
// -------------

func SecondSecond(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	date := c.Param("date")
	place := c.Param("place")

	rows, err := db.Query("select date, name_jp, npatients from infection where name_jp = ? and date like ? ORDER BY date ASC", place, date+"%")
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

// -------------
// 2 - 3
// -------------

func SecondThird(c *gin.Context) {
	// Connect to the database
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	date := c.Param("date")
	place := c.Param("place")

	rows, err := db.Query("select date, name_jp, npatients from infection where name_jp = ? and date like ? order by date ASC", place, date+"%")
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

// -------------
// 3 - 1
// -------------

func Create(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var json Event_JSON
	validate := Validate() //インスタンス生成

	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// yyyymmdd形式の文字列をtime.Time型に変換
	layout := "20060102"
	t, err := time.Parse(layout, json.Begin)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// yyyymmdd形式の文字列をtime.Time型に変換
	t2, err := time.Parse(layout, json.End)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err = validate.Struct(&json); err != nil { //バリデーションを実行し、NGの場合、ここでエラーが返る。
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	insert, err := db.Prepare("INSERT INTO events (title, description, begin, end) VALUES (?, ?, ?, ?)")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer insert.Close()

	// パラメータを設定してクエリを実行
	_, err = insert.Exec(json.Title, json.Description, t.Format("2006-01-02"), t2.Format("2006-01-02"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func Show(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
		return
	}
	defer db.Close()

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}) // 400
		return
	}

	var json Event_JSON
	err = db.QueryRow("SELECT title, description, begin, end FROM events WHERE id = ?", id).Scan(&json.Title, &json.Description, &json.Begin, &json.End)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "event not found"}) // 404
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, json)
}

func ShowAll(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT title, description, begin, end FROM events")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
		return
	}
	defer rows.Close()

	var result []Event_JSON
	for rows.Next() {
		var json Event_JSON
		if err := rows.Scan(&json.Title, &json.Description, &json.Begin, &json.End); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
			return
		}
		result = append(result, json)
	}

	c.JSON(http.StatusOK, result) // 200
}

func Update(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
		return
	}
	defer db.Close()

	var json Event_JSON
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"}) // 400
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}) // 400
		return
	}

	update, err := db.Prepare("UPDATE events SET title = ?, description = ?, begin = ?, end = ? WHERE id = ?")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
		return
	}
	if _, err := update.Exec(json.Title, json.Description, json.Begin, json.End, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
		return
	}

	c.Status(http.StatusOK) // 200
}

func Delete(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
		return
	}
	defer db.Close()

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}) // 400
		return
	}

	delete, err := db.Prepare("DELETE FROM events WHERE id = ?")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
		return
	}
	if _, err := delete.Exec(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
		return
	}

	c.Status(http.StatusOK) // 200
}

// -------------
// 3 - 2
// -------------

func ThirdSecond(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	date1 := c.Param("date1")
	date2 := c.Param("date2")

	rows, err := db.Query("select date, name_jp, npatients from infection where date between ? and ? order by date ASC", date1, date2)
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

// -------------
// 3 - 3
// -------------

func ThirdThird(c *gin.Context) {
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

func ForthFirst(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	place := c.Param("place")

	rows, err := db.Query("select facility_name, facility_addr, facility_type from medical where pref_name = ?", place)
	if err != nil {
		log.Fatal(err)
	}
	var resultMedical []Medicals

	for rows.Next() {
		medical := Medicals{}
		if err := rows.Scan(&medical.FacilityName, &medical.FacilityAddr, &medical.FacilityType); err != nil {
			log.Fatal(err)
		}
		resultMedical = append(resultMedical, medical)
	}

	c.JSON(http.StatusOK, resultMedical)
}

func Validate() *validator.Validate {
	validate := validator.New()
	return validate
}

func Import(c *gin.Context) {
	log.Print("データ取り込み中")
	url := "https://opendata.corona.go.jp/api/Covid19JapanAll"
	resp, _ := http.Get(url)
	if resp != nil {
		defer resp.Body.Close()
	}
	byteArray, _ := ioutil.ReadAll(resp.Body)

	jsonBytes := ([]byte)(byteArray)
	data := new(Npatients)

	if err := json.Unmarshal(jsonBytes, data); err != nil {
		fmt.Println("JSON Unmarshal error:", err)
		return
	}

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

func ImportMedical(c *gin.Context) {
	log.Print("データ取り込み中")
	// JSONデータを取得する
	resp, err := http.Get("https://opendata.corona.go.jp/api/covid19DailySurvey")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	byteArray, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	var records []Medical
	if err := json.Unmarshal(byteArray, &records); err != nil {
		panic(err)
	}

	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	delete, err := db.Prepare("DELETE FROM medical")
	if err != nil {
		log.Fatal(err)
	}
	delete.Exec()

	insert, err := db.Prepare("INSERT INTO medical (facility_id, facility_name, zip_code, pref_name, facility_addr, facility_tel, latitude, longitude, submit_date, facility_type, ans_type, local_gov_code, city_name, facility_code) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer insert.Close()

	for _, f := range records {
		_, err = insert.Exec(f.FacilityId, f.FacilityName, f.ZipCode, f.PrefName, f.FacilityAddr, f.FacilityTel, f.Latitude, f.Longitude, f.SubmitDate, f.FacilityType, f.AnsType, f.LocalGovCode, f.CityName, f.FacilityCode)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.Status(http.StatusOK)

	log.Print("データ取り込み完了")
}
