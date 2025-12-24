package main

import (
	"fmt"
	"log"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// --- 1. KONFIGURASI DATABASE & MODEL ---

var db *gorm.DB
var secretKey = []byte("rahasia_gdgoc_anjelia") // Kunci rahasia untuk token

// Definisi Tabel User
type User struct {
	gorm.Model
	Email    string `gorm:"unique"`
	Password string
	Role     string // "admin" atau "user"
}

// Definisi Tabel Event
type Event struct {
	gorm.Model
	Name  string
	Stock int
}

// Definisi Tabel Transaksi
type Transaction struct {
	gorm.Model
	UserID  uint
	EventID uint
	Qty     int
}

// Fungsi Koneksi Database (Otomatis buat file ticket.db)
func connectDB() {
	var err error
	db, err = gorm.Open(sqlite.Open("ticket.db"), &gorm.Config{})
	if err != nil {
		panic("Gagal konek database")
	}

	// Auto Migrate: Membuat tabel otomatis
	db.AutoMigrate(&User{}, &Event{}, &Transaction{})

	// SEEDING: Isi data awal otomatis jika kosong
	var count int64
	db.Model(&User{}).Count(&count)
	if count == 0 {
		// Buat 1 Admin dan 1 User
		db.Create(&User{Email: "admin@unsri.ac.id", Password: "admin", Role: "admin"})
		db.Create(&User{Email: "user@unsri.ac.id", Password: "user", Role: "user"})
		// Buat 1 Event
		db.Create(&Event{Name: "GDGoC Festival", Stock: 10})
		fmt.Println(">> Data Dummy Dibuat: Admin(admin/admin), User(user/user)")
	}
}

// --- 2. MIDDLEWARE (SATPAM PENCEGAH PENYUSUP) ---

// AuthMiddleware: Cek apakah user punya Token JWT?
func authRequired(c *fiber.Ctx) error {
	// Ambil token dari header
	tokenString := c.Get("Authorization")
	if tokenString == "" {
		return c.Status(401).JSON(fiber.Map{"error": "Anda belum login (Butuh Token)"})
	}

	// Cek validitas token
	token, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})

	// Jika valid, simpan data user ke context (biar bisa dipakai nanti)
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		c.Locals("user_id", claims["user_id"])
		c.Locals("role", claims["role"])
		return c.Next() // Lanjut boleh masuk
	}

	return c.Status(401).JSON(fiber.Map{"error": "Token salah atau kadaluarsa"})
}

// AdminOnly: Cuma admin yang boleh lewat
func adminOnly(c *fiber.Ctx) error {
	role := c.Locals("role")
	if role != "admin" {
		return c.Status(403).JSON(fiber.Map{"error": "Akses Ditolak: Khusus Admin"})
	}
	return c.Next()
}

// --- 3. CONTROLLER (LOGIKA BISNIS) ---

// Login: Tukar Email & Password jadi Token
func login(c *fiber.Ctx) error {
	type LoginInput struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	var input LoginInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).SendString("Input salah")
	}

	// Cek di Database
	var user User
	if err := db.Where("email = ? AND password = ?", input.Email, input.Password).First(&user).Error; err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "Email atau Password salah"})
	}

	// Bikin Token JWT
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     time.Now().Add(time.Hour * 24).Unix(), // Token berlaku 24 jam
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t, _ := token.SignedString(secretKey)

	return c.JSON(fiber.Map{"token": t, "role": user.Role})
}

// CreateEvent: Admin bikin acara
func createEvent(c *fiber.Ctx) error {
	event := new(Event)
	if err := c.BodyParser(event); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Input invalid"})
	}
	db.Create(&event)
	return c.JSON(fiber.Map{"message": "Event berhasil dibuat", "data": event})
}

// BookTicket: User beli tiket (Pakai Transaction DB biar aman)
func bookTicket(c *fiber.Ctx) error {
	type BookingInput struct {
		EventID uint `json:"event_id"`
		Qty     int  `json:"qty"`
	}
	var input BookingInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).SendString("Input salah")
	}

	userID := c.Locals("user_id").(float64) // Ambil ID user dari Token

	// === TRANSAKSI DATABASE (Secure the Crowd Logic) ===
	// Kita pakai db.Transaction. Jika ada error di tengah, semua batal (Rollback).
	err := db.Transaction(func(tx *gorm.DB) error {
		var event Event

		// LOCKING: Kunci baris data ini di database supaya tidak direbut user lain
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&event, input.EventID).Error; err != nil {
			return err // Event gak ketemu
		}

		// Cek Stok
		if event.Stock < input.Qty {
			return fmt.Errorf("stok habis") // Bikin error manual buat cancel transaksi
		}

		// Kurangi Stok
		event.Stock -= input.Qty
		if err := tx.Save(&event).Error; err != nil {
			return err
		}

		// Catat Transaksi
		booking := Transaction{
			UserID:  uint(userID),
			EventID: input.EventID,
			Qty:     input.Qty,
		}
		if err := tx.Create(&booking).Error; err != nil {
			return err
		}

		return nil // Sukses
	})
	// === SELESAI TRANSAKSI ===

	if err != nil {
		return c.Status(409).JSON(fiber.Map{"status": "Gagal", "error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "Berhasil", "message": "Tiket diamankan!"})
}

// --- 4. MAIN PROGRAM ---
func main() {
	connectDB() // Hubungkan database

	app := fiber.New()
	app.Use(logger.New()) // Biar kelihatan log di terminal

	// Route Publik (Bisa diakses siapa saja)
	app.Post("/login", login)

	// Route Terproteksi (Harus punya Token)
	api := app.Group("/api", authRequired)

	// User Routes
	api.Post("/book", bookTicket)

	// Admin Routes
	api.Post("/events", adminOnly, createEvent)

	fmt.Println("Server Secure Crowd berjalan di port 3000...")
	log.Fatal(app.Listen(":3000"))
}
