# OpenFlux — Маскированный сетевой туннель и Android-клиент

[English](README.md) | **Русский**

**OpenFlux** — система защищённой маскированной передачи сетевого TCP-трафика через легитимные облачные сервисы (в текущей конфигурации — через веб-сокет совместного редактирования Яндекс.Документов и WebRTC DataChannel мессенджера MAX). Включает высокопроизводительную выходную ноду (Exit Node) на Go, десктопный SOCKS5-клиент и нативное **Android-приложение** с поддержкой полносистемного VPN и раздельного туннелирования (Split Tunneling).

---

## Архитектура системы

```
┌─────────────────────────────────────────────────────────┐
│                    Клиентская часть                     │
│  [Android App / Браузер] ──> SOCKS5 Proxy (:1080)       │
│                                  │                      │
│                  Сжатие полезной нагрузки LZ4           │
│                                  │                      │
│           Сквозное шифрование ChaCha20-Poly1305         │
└──────────────────────────────┬──────────────────────────┘
                               │ (Маскированный WebSocket / WebRTC)
                               ▼
┌─────────────────────────────────────────────────────────┐
│               Облачный сервис-посредник                 │
│      (Яндекс.Документы WebSocket / WebRTC MAX)          │
│       * Посредник видит лишь зашифрованный шум *        │
└──────────────────────────────┬──────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────┐
│                   Выходная нода (VDS)                   │
│             Дешифрование ChaCha20-Poly1305              │
│                                  │                      │
│                  Декомпрессия полезной нагрузки LZ4     │
│                                  │                      │
│       Сырые сокеты Linux (tunnel/rawsocket_linux.go)    │
│                                  │                      │
│                        Целевой Интернет                 │
└─────────────────────────────────────────────────────────┘
```

---

## Основные возможности

- **Подключаемые маскирующие транспорты:**
  - **Яндекс.Документы:** эмуляция сессии совместного редактирования через Socket.IO v4 поверх WebSocket. Сетевые пакеты инкапсулируются в сообщения перемещения курсора и ревизии документа.
  - **Мессенджер MAX:** передача трафика через сигнальный протокол и WebRTC DataChannel.
- **Сквозное AEAD-шифрование ChaCha20-Poly1305:**
  - Zero-Knowledge передача: серверы облака (Яндекс, MAX) не могут перехватить заголовки, домены (SNI), URL и DNS-запросы.
  - Индивидуальный 12-байтный случайный Nonce на каждый пакет + 16-байтный аутентификационный тег Poly1305.
  - Настройка ключа через интерфейс приложения или параметр запуска `-key`.
- **Полнофункциональное Android-приложение (`android/`):**
  - **Современный Material 3 интерфейс:** поддержка тёмной и светлой темы.
  - **Режим «Только прокси»:** компактный фоновый сервис, поднимающий локальный SOCKS5 (`127.0.0.1:1080` или `0.0.0.0:1080` для раздачи в локальную сеть).
  - **Режим VPN:** Android `VpnService`, перехватывающий весь трафик устройства в виртуальный интерфейс `tun0` через стек `tun2socks` (gVisor).
  - **Раздельное туннелирование (Split Tunneling):** белый и чёрный списки приложений, переключатель системных программ.
  - **DNS-over-TCP:** автоматический перехват Android UDP DNS-запросов (53 порт) и упаковка в RFC 1035 TCP-туннель.
  - **Живой мониторинг трафика:** отображение объёма и скорости на главном экране и в шторке уведомлений Android (`↑ / ↓`).
  - **Защита от сбоев:** перехват паник ядра и встроенный просмотрщик логов (`LogActivity`).
- **Надёжная серверная часть для Linux (Exit Node):**
  - Маршрутизация трафика через сырые сокеты (`AF_INET RAW`).
  - Автоматическое отслеживание активных клиентских портов и таймеры очистки.
  - Принудительный IPv4 (`tcp4`), исключающий зависания соединений на dual-stack серверах.
- **Надёжность и безопасность:**
  - Устранены уязвимости отказа в обслуживании (DoS), введены лимиты на аллокацию памяти, защита от Decompression Bomb в LZ4, потокобезопасная синхронизация.

---

## Структура проекта

```
OpenFlux/
├── main.go                     # Точка входа CLI (десктопный клиент и серверная нода)
├── transport/
│   ├── transport.go            # Базовый интерфейс Transport
│   ├── encrypted.go            # Сквозное AEAD-шифрование ChaCha20-Poly1305
│   ├── compressor.go           # Сжатие LZ4 с защитой от Decompression Bomb
│   ├── yandex/                 # Транспорт Яндекс.Документов (WebSocket)
│   └── oneme/                  # Транспорт мессенджера MAX (WebRTC)
├── tunnel/
│   ├── tunnel.go               # Сетевой стек gVisor и маршрутизация
│   ├── endpoint.go             # Виртуальный интерфейс LinkEndpoint
│   └── rawsocket_linux.go      # Сырые сокеты Linux и трекинг портов
├── socks5/
│   └── socks5.go               # SOCKS5-сервер и DNS-over-TCP (RFC 1035)
├── network/
│   └── checksum.go             # Пересчёт контрольных сумм IP и TCP
├── mobile/
│   └── bridge.go               # JNI C-Shared мост между Go core и Android ART
├── android/                    # Android Studio проект (Kotlin + Jetpack)
│   ├── app/src/main/           # Исходный код Android приложения
│   └── openflux-release.jks    # Ключ подписи релизных APK
├── releases/                   # Папка для готовых релизных сборок
├── scripts/
│   └── build_android_lib.ps1   # Сборка Go-библиотеки через NDK Clang
├── build_release.bat           # Сборка Release APK в 1 клик для Windows
└── build_apk.ps1               # Сборка Debug APK
```

---

## Быстрый старт

### 1. Настройка выходной ноды (Linux VPS)

Для работы ноды выхода требуются права root (для сырых сокетов).

```bash
# 1. Сборка бинарника для Linux
go build -ldflags="-s -w" -o universal-bypass-tool .

# 2. Подавление сброса TCP-пакетов ядром Linux
sudo iptables -I OUTPUT 1 -p tcp --tcp-flags RST RST -j DROP

# 3. Запуск выходной ноды
sudo ./universal-bypass-tool -exit-node \
  -transport yandex \
  -url "https://disk.yandex.ru/i/ВАШ_КЛЮЧ_ДОКУМЕНТА" \
  -key "ВАШ_КЛЮЧ_ШИФРОВАНИЯ" \
  -debug
```

#### Запуск через Systemd-сервис

Создайте файл `/etc/systemd/system/openflux-exit.service`:
```ini
[Unit]
Description=OpenFlux Exit Node
After=network.target network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/openflux
ExecStartPre=/bin/sh -c '/sbin/iptables -C OUTPUT -p tcp --tcp-flags RST RST -j DROP 2>/dev/null || /sbin/iptables -I OUTPUT 1 -p tcp --tcp-flags RST RST -j DROP'
ExecStart=/opt/openflux/universal-bypass-tool -exit-node -transport yandex -url "https://disk.yandex.ru/i/ВАШ_ДОКУМЕНТ" -debug
ExecStopPost=/sbin/iptables -D OUTPUT -p tcp --tcp-flags RST RST -j DROP
Restart=always
RestartSec=5s
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
```
Активируйте и запустите службу:
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now openflux-exit.service
```

---

### 2. Сборка Android-клиента

#### Автоматическая сборка Release APK в 1 клик (Windows):
Запустите [build_release.bat](build_release.bat):
```cmd
build_release.bat
```
Скрипт автоматически:
1. Скомпилирует `libopenflux.so` со всеми оптимизациями (`-trimpath`, `-ldflags="-s -w"`, CGO `-O3`).
2. Запустит Gradle `assembleRelease` с **R8-обфускацией/минификацией** и **Resource Shrinking**.
3. Подпишет APK схемой **APK Signature Scheme v2**.
4. Сохранит готовый к установке пакет в папку `releases/OpenFlux-release.apk` (размер ~17 МБ).

#### Ручная сборка через PowerShell:
```powershell
# Сборка Go-библиотеки под arm64-v8a
$env:GOOS = "android"; $env:GOARCH = "arm64"; $env:CGO_ENABLED = "1"
$env:CC = "C:\Users\<user>\AppData\Local\Android\Sdk\ndk\<версия>\toolchains\llvm\prebuilt\windows-x86_64\bin\aarch64-linux-android24-clang.cmd"
go build -trimpath -buildmode=c-shared -ldflags="-s -w -checklinkname=0" -o android/app/src/main/jniLibs/arm64-v8a/libopenflux.so ./mobile

# Сборка APK
cd android
.\gradlew.bat assembleRelease
```

---

### 3. Десктопный клиент (CLI)

Для маршрутизации трафика браузера или системы через туннель на ПК:

```bash
./universal-bypass-tool -client \
  -transport yandex \
  -url "https://disk.yandex.ru/i/ВАШ_КЛЮЧ_ДОКУМЕНТА" \
  -socks5 127.0.0.1:1080 \
  -key "ВАШ_КЛЮЧ_ШИФРОВАНИЯ" \
  -debug
```
После запуска укажите SOCKS5 прокси `127.0.0.1:1080` в настройках вашей системы или браузера.

---

## Флаги командной строки

| Флаг | По умолчанию | Описание |
|---|---|---|
| `-client` | `false` | Запуск в режиме клиента (поднимает SOCKS5) |
| `-exit-node` | `false` | Запуск в режиме сервера выхода (требует root) |
| `-transport` | `yandex` | Выбор транспорта: `yandex` или `oneme` |
| `-url` | `http://#` | Публичная ссылка на документ Яндекс.Диска |
| `-socks5` | `:1080` | Адрес прослушивания SOCKS5-сервера |
| `-key` | *(встроенный)* | Ключ ChaCha20-Poly1305 (или переменная `OPENFLUX_KEY`) |
| `-maxToken` | `""` | Токен авторизации мессенджера MAX |
| `-maxUid` | `""` | Идентификатор пользователя MAX |
| `-debug` | `false` | Включение подробного диагностического вывода |

---

## Безопасность и приватность

1. **Защита от сервера-посредника:** Облачный сервис видит лишь base64-строки курсорных сообщений с высокой энтропией (криптографический белый шум). Никакие сетевые заголовки, домены и данные не утекают в сервис Яндекса или MAX.
2. **Защита от Zip-Bomb:** Распаковка LZ4 ограничена константой `MaxDecompressedSize = 10 MB` на фрейм, предотвращая атаки на исчерпание оперативной памяти.
3. **Безопасная архитектура:** Защита от переполнения буферов, перехват паник во всех горутинах и строгий контроль границ пакетов.

---

## Лицензия

Проект разработан исключительно в исследовательских и образовательных целях для изучения протоколов маскирования и устойчивой передачи данных. Используйте только на подконтрольных вам серверах и сетях.
