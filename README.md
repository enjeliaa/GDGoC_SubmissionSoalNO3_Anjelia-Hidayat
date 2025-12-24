# Soal 3: Studi Kasus II (Secure the Crowd!)
**Backend Development Submission - GDGoC**
**Created by: Anjelia Hidayat**

Ini adalah solusi untuk tantangan backend "Secure the Crowd". Aplikasi ini dibangun menggunakan **Golang**, **Fiber**, **GORM**, dan **JWT Authentication**.

## Fitur Utama
1.  **JWT Authentication:** Login aman menggunakan token.
2.  **Role Based Access:** Membedakan hak akses antara User dan Admin.
3.  **Transaction & Locking:** Mencegah race condition saat pembelian tiket (stok aman).
4.  **Database:** Menggunakan SQLite (File `ticket.db` akan dibuat otomatis).

## Cara Menjalankan Program
1.  Pastikan Golang sudah terinstall.
2.  Buka terminal di folder project ini.
3.  Download library yang dibutuhkan:
    ```bash
    go mod tidy
    ```
4.  Jalankan server:
    ```bash
    go run main.go
    ```
5.  Server akan berjalan di `http://localhost:3000`.

---

## Dokumentasi API (Cara Testing)

Gunakan **Postman** untuk pengujian.

### 1. Login (Mendapatkan Token)
Endpoint ini digunakan untuk masuk dan mendapatkan "kunci" (Token) agar bisa akses fitur lain.

* **URL:** `http://localhost:3000/login`
* **Method:** `POST`
* **Body (Raw JSON):**
    ```json
    {
        "email": "user@unsri.ac.id",
        "password": "user"
    }
    ```
* **Response Sukses:**
    Akan mengembalikan JSON berisi `token`. **Copy token tersebut** untuk langkah selanjutnya.

---

### 2. Membeli Tiket (Booking)
Hanya bisa diakses jika menyertakan Token di Header.

* **URL:** `http://localhost:3000/api/book`
* **Method:** `POST`
* **Authorization (PENTING):**
    * Pilih tab **Headers** di Postman.
    * Key: `Authorization`
    * Value: `(Paste Token Panjang Disini)`
* **Body (Raw JSON):**
    ```json
    {
        "event_id": 1,
        "qty": 2
    }
    ```
* **Response Sukses:**
    ```json
    {
        "status": "Berhasil",
        "message": "Tiket diamankan!"
    }
    ```

---

### 3. Tambah Event (Khusus Admin)
Hanya bisa dilakukan oleh akun Admin.

* **URL:** `http://localhost:3000/api/events`
* **Method:** `POST`
* **Login Admin Dulu:** Gunakan email `admin@unsri.ac.id` dan password `admin` di menu Login untuk dapat token admin.
* **Headers:** Masukkan token admin di Header `Authorization`.
* **Body (Raw JSON):**
    ```json
    {
        "name": "GDGoC Bootcamp 2025",
        "stock": 100
    }
    ```

---
*Terima kasih.*