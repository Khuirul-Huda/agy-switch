<div align="center">

# 🔄 agy-switch

**Kakas baris perintah (CLI) yang cepat, aman, dan tanpa overhead untuk berganti multi-profil akun pada Antigravity CLI (`agy`).**

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go)](https://golang.org)
[![Platform](https://img.shields.io/badge/Platforms-Linux%20%7C%20macOS%20%7C%20Windows-555555?style=for-the-badge&logo=linux&logoColor=white)](https://github.com/Khuirul-Huda/agy-switch)
[![Architecture](https://img.shields.io/badge/Architecture-Clean%20Hexagonal-8A2BE2?style=for-the-badge)](https://github.com/Khuirul-Huda/agy-switch)
[![Tests](https://img.shields.io/badge/Tests-Hermetic%20In--Memory-brightgreen?style=for-the-badge&logo=githubactions&logoColor=white)](https://github.com/Khuirul-Huda/agy-switch)
[![License](https://img.shields.io/badge/License-MIT-blue?style=for-the-badge)](LICENSE)

Beralih antara akun Google Pribadi, Kantor, dan Klien di `agy` secara instan tanpa perlu membuka peramban web atau melakukan login ulang setiap hari.

---

📖 **[ 🇬🇧 Read in English ](README.md)**

---

</div>

> [!CAUTION]
> ### ⚠️ PERINGATAN PENTING & DISCLAIMER
> - **Gunakan dengan risiko Anda sendiri (*Use at your own risk*):** Perangkat lunak ini disediakan "sebagaimana adanya" (*as is*), tanpa jaminan apa pun baik tersurat maupun tersirat. Penulis dan kontributor tidak bertanggung jawab atas klaim, kerugian, pencabutan token, atau tindakan terkait akun yang timbul akibat penggunaan kakas ini.
> - **Proyek Independen / Tidak Resmi:** `agy-switch` adalah kakas sumber terbuka independen yang dikembangkan oleh komunitas. Kakas ini **TIDAK berafiliasi dengan, disponsori oleh, diotorisasi oleh, didukung oleh, atau memiliki hubungan resmi apa pun dengan Google LLC, Google DeepMind, Alphabet Inc., maupun anak perusahaannya.**
> - "Google", "Antigravity", dan "Gemini" adalah merek dagang terdaftar milik Google LLC.

---

## 💡 Masalah yang Diselesaikan

Secara bawaan, Google Antigravity CLI (`agy`) hanya mengikat satu akun Google yang tersimpan di sistem keyring OS. Jika Anda memiliki akun Google pribadi dan akun Google Cloud kantor/kampus:
- Anda harus mengetik `/logout`.
- Menunggu browser terbuka untuk login ulang.
- Menekan tombol persetujuan izin berulang-ulang setiap hari.

## ✨ Solusi dengan `agy-switch`

`agy-switch` mengelola multi-profil akun langsung di dalam penyimpanan kredensial aman bawaan OS Anda (**Linux Secret Service / KWallet / GNOME Keyring**, **macOS Apple Keychain**, dan **Windows Credential Manager**).

- 🔒 **Tanpa File Teks Polos:** Kredensial tidak pernah disimpan di hard disk dalam bentuk file JSON polos. Semua data terenkripsi aman di OS Keyring bawaan perangkat Anda.
- ⚡ **Peralihan Instan:** Berganti akun aktif dalam hitungan milidetik tanpa interaksi browser.
- 🖥️ **Menu Interaktif & Perintah Cepat:** Jalankan `agy-switch` tanpa argumen untuk menu interaktif, atau gunakan subcommand untuk otomatisasi.
- 🌐 **Mendukung Penuh Linux, macOS, dan Windows.**

---

## 🖥️ Contoh Tampilan (Demo)

```text
$ agy-switch

=== agy-switch : Antigravity Profile Switcher ===
Active account: personal@gmail.com

NO   PROFILE          EMAIL                            STATUS    
─────────────────────────────────────────────────────────────────
 1)  personal         personal@gmail.com               ● ACTIVE  
 2)  work             developer@company.com                      

Select profile number to switch (1-2, or 'q' to quit): 2
✓ Successfully switched to profile 'work' (developer@company.com)!
Launch 'agy' in a new terminal session to start working with this account.
```

---

## 🚀 Panduan Cepat (Quick Start)

### 1. Simpan sesi aktif saat ini:
```bash
agy-switch save personal
```

### 2. Hubungkan akun kedua (Kantor / Kampus / Klien):
```bash
agy-switch logout
agy   # agy akan membuka browser untuk login akun kedua
```
Setelah login di browser sukses, simpan profil baru tersebut:
```bash
agy-switch save work
```

### 3. Pindah akun kapan saja:
```bash
# Menu interaktif:
agy-switch

# Atau perintah langsung:
agy-switch switch work
agy-switch switch personal
```

---

## 📦 Pemasangan (Installation)

### Biner Siap Pakai (Pre-compiled)

Unduh biner sesuai sistem operasi Anda dari halaman [Releases](https://github.com/Khuirul-Huda/agy-switch/releases):

#### Linux (Debian, Ubuntu, Arch, Fedora, openSUSE)
```bash
chmod +x agy-switch-linux-amd64
mv agy-switch-linux-amd64 ~/.local/bin/agy-switch
```

#### macOS (Apple Silicon M1/M2/M3/M4 & Intel)
```bash
# Apple Silicon:
chmod +x agy-switch-darwin-arm64
sudo mv agy-switch-darwin-arm64 /usr/local/bin/agy-switch

# Intel Mac:
chmod +x agy-switch-darwin-amd64
sudo mv agy-switch-darwin-amd64 /usr/local/bin/agy-switch
```

#### Windows
Ubah nama `agy-switch-windows-amd64.exe` menjadi `agy-switch.exe` dan masukkan ke direktori yang ada di `PATH` sistem Anda.

---

### Kompilasi dari Source (Membutuhkan Go 1.21+)

```bash
git clone git@github.com:Khuirul-Huda/agy-switch.git
cd agy-switch

# Build untuk mesin saat ini:
go build -o bin/agy-switch main.go

# Pasang ke PATH lokal:
cp bin/agy-switch ~/.local/bin/
```

---

## 📖 Ringkasan Perintah

| Perintah | Alias | Penjelasan |
|---|---|---|
| `agy-switch` | — | Membuka menu interaktif pemilihan profil |
| `agy-switch current` | — | Menampilkan email akun yang sedang aktif di `agy` |
| `agy-switch save <nama>` | — | Menyimpan sesi aktif ke profil bernama |
| `agy-switch switch <nama>` | — | Mengaktifkan profil tersimpan ke `agy` |
| `agy-switch list` | `ls` | Menampilkan tabel profil yang tersimpan |
| `agy-switch delete <nama>` | `rm` | Menghapus profil dari OS Keyring |
| `agy-switch logout` | — | Menghapus sesi aktif agar `agy` meminta login baru |
| `agy-switch version` | `-v` | Menampilkan informasi versi |
| `agy-switch help` | `-h` | Menampilkan bantuan penggunaan |

---

## 🔒 Keamanan & Arsitektur

`agy-switch` berkomunikasi dengan penyimpanan kredensial yang sama persis dengan yang dipakai oleh biner resmi Google `agy`:

```
┌────────────────────────────────────────────────────────┐
│                   agy-switch CLI                       │
└──────────────────────────┬─────────────────────────────┘
                           │
                 [ Keyring Storage Port ]
                           │
       ┌───────────────────┼───────────────────┐
       ▼                   ▼                   ▼
 Linux Secret Service  macOS Keychain   Windows Credential
 (KWallet / GNOME)    (Security API)         Manager
       │                   │                   │
       └───────────────────┼───────────────────┘
                           ▼
          Namespace: service = "gemini"
          • Active Token : username = "antigravity"
          • Saved Profile: username = "antigravity:profile:<name>"
          • Registry     : username = "antigravity:_profiles"
```

1. **Keamanan Bawaan OS:** Token dilindungi oleh password login sistem operasi / modul PAM / autentikasi biometrik.
2. **Auto-Refresh OAuth Standar:** Saat berpindah ke profil lama yang *access token*-nya telah kedaluwarsa, `agy` akan memanfaatkan `refresh_token` di dalam profil tersebut untuk meminta token baru dari server Google secara otomatis.
3. **Hermetis & Aman:** Kakas ini tidak menyentuh database histori atau berkas proyek lokal, melainkan hanya menukar *pointer* kredensial sesi aktif.

---

## 🏛️ Clean Architecture & Testing

Proyek ini mematuhi prinsip **Clean Code**, **SOLID**, dan **Hexagonal Architecture (*Ports & Adapters*)**:

```text
internal/
├── domain/            # Model bisnis murni & validasi (0 dependensi eksternal)
│   ├── profile.go
│   └── profile_test.go
├── port/              # Kontrak interface
│   └── keyring.go     # KeyringStorage port
├── service/           # Use cases aplikasi (Save, Switch, List, Delete)
│   ├── profile_service.go
│   └── profile_service_test.go
└── adapter/           # Infrastruktur & Delivery
    ├── system_keyring.go  # Adapter OS Keyring ril
    ├── memory_keyring.go  # In-Memory test double (zero disk touch)
    └── cli_ui.go          # ANSI terminal rendering & interactive prompt
```

### 🧪 Menjalankan Unit Test (Hermetic In-Memory)
Seluruh pengujian unit berjalan di atas `MemoryKeyring` di memori RAM dan **sama sekali tidak menyentuh Keyring asli sistem operasi**:

```bash
go test -v -race ./...
```

---

## 📄 Lisensi

Didistribusikan di bawah lisensi **MIT License**. Lihat berkas `LICENSE` untuk informasi lebih lanjut.
