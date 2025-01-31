package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// สร้าง Fiber App
	app := fiber.New()

	
	app.Use(cors.New())

	// ตั้งค่า DSN (Data Source Name) สำหรับ MySQL
	dsn := "root:mint392488@tcp(127.0.0.1:3306)/golang_dbmysql"

	// ตั้งค่า Context สำหรับการเชื่อมต่อ (Timeout 5 วินาที)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// เชื่อมต่อ MySQL
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("ไม่สามารถเปิดการเชื่อมต่อ MySQL:", err)
	}

	// ตั้งค่าการเชื่อมต่อ (Connection Pool)
	db.SetConnMaxLifetime(10 * time.Minute)
	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(10)

	// ทดสอบการเชื่อมต่อ
	err = db.PingContext(ctx)
	if err != nil {
		log.Fatal("เชื่อมต่อฐานข้อมูล MySQL ล้มเหลว:", err)
	}

	fmt.Println("เชื่อมต่อฐานข้อมูล MySQL สำเร็จ")

	// โครงสร้างข้อมูล Breed
	type Breed struct {
		ID         string    `json:"id"`
		NameEn     string    `json:"name_en"`
		NameTh     string    `json:"name_th"`
		ShortName  string    `json:"short_name"`
		Remark     *string   `json:"remark"`
		//CreatedAt  time.Time `json:"created_at"`
		//CreatedBy  string    `json:"created_by"`
		//CreatedByID  string    `json:"created_by_id"`
		//UpdatedAt  time.Time `json:"updated_at"`
		//UpdatedBy  string    `json:"updated_by"`
		//UpdatedByID  string    `json:"updated_by_id"`
	}

	// API `/api/breed-inquiry`
	app.Post("/api/breed-inquiry", func(c *fiber.Ctx) error {
		// อ่านค่า Request
		type Request struct {
			IDs        []string `json:"ids"`
			Keyword    string   `json:"keyword"`    
			ShortNames []string `json:"shortnames"` 
		}

		req := new(Request)
		if err := c.BodyParser(req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
		}

		// ใช้ context.WithTimeout() เพื่อควบคุมเวลาของ Query (Timeout 3 วินาที)
		queryCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		// สร้างเงื่อนไขการค้นหาแบบ Dynamic
		//query := "SELECT * FROM breed WHERE 1=1"
		query := "SELECT id, name_th, name_en, short_name, remark FROM breed WHERE 1=1"
		var args []interface{}

		// กรองตาม Keyword
		if req.Keyword != "" {
			query += " AND (name_th LIKE ? OR name_en LIKE ?)"
			args = append(args, "%"+req.Keyword+"%", "%"+req.Keyword+"%")
		}

		// กรองตาม IDs
		if len(req.IDs) > 0 {
			query += " AND id IN (?" + strings.Repeat(",?", len(req.IDs)-1) + ")"
			for _, id := range req.IDs {
				args = append(args, id)
			}
		}

		// กรองตาม ShortNames
		if len(req.ShortNames) > 0 {
			query += " AND short_name IN (?" + strings.Repeat(",?", len(req.ShortNames)-1) + ")"
			for _, sn := range req.ShortNames {
				args = append(args, sn)
			}
		}
        fmt.Println("Query:", query)
		fmt.Println("Args:", args)

		// ค้นหาข้อมูลจาก Database
		rows, err := db.QueryContext(queryCtx, query, args...)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Database error"})
		}
		defer rows.Close()

		fmt.Println("Query executed successfully, rows:", rows)
		// เก็บผลลัพธ์ทั้งหมด
		var breeds []Breed
		for rows.Next() {
			fmt.Println("Found a row") 
			var breed Breed
			if err := rows.Scan(&breed.ID, &breed.NameEn, &breed.NameTh, &breed.ShortName, &breed.Remark); err != nil {
				//, &breed.CreatedAt, &breed.CreatedBy, &breed.CreatedByID, &breed.UpdatedAt, &breed.UpdatedBy, &breed.UpdatedByID
				fmt.Println("Error scanning data:", err) 
				return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Error scanning data"})
			}
			breeds = append(breeds, breed)
		}
		fmt.Println("Query executed successfully, rows:", rows)



		// ถ้าไม่พบข้อมูล
		if len(breeds) == 0 {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"status": "error", "message": "Breed not found"})
		}

		// ส่งข้อมูลกลับไป
		return c.JSON(fiber.Map{
			"status": "success",
			"data":   breeds,
		})
	})

	// ทดสอบ API
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "running"})
	})

	// เริ่มเซิร์ฟเวอร์
	log.Fatal(app.Listen(":3000"))
}

