# OpenFlux — Сетевой туннель, Windows GUI и Android-клиент

[English](README.md) | **Русский**

OpenFlux — исследовательский инструмент сетевого стека: TCP/UDP-туннель с подключаемыми транспортами и сквозным шифрованием (E2E).

Форк оригинального репозитория [p1neappleXpress/OpenFlux](https://github.com/p1neappleXpress/OpenFlux).

> [!NOTE]
> **Отказ от ответственности**: Проект является некоммерческим исследовательским инструментом и предоставляется «как есть» исключительно в образовательных целях для изучения работы сетевых протоколов и виртуальных сетевых интерфейсов.

---

## Возможности

- **Подключаемые транспорты (`-transport`):**
  - `yandex` — сессия совместного редактирования Яндекс Документов (Socket.IO v4 поверх WebSocket).
  - `vyandex` — потоковый HTTP-транспорт Яндекс.Волга.
  - `mailru` — туннелирование трафика через сессию совместного редактирования документов Mail.ru Cloud (`docs.datacloudmail.ru`).
  - `cupsonline` — транспорт поверх совместных сессий Cups.online (WebSocket).
  - `oneme` — передача трафика через сигнальный протокол и WebRTC DataChannel мессенджера MAX.
- **Пакетный кодек и сжатие (Batched Codec):**
  - По умолчанию включён высокоэффективный пакетный кодек (`--codec=batched`): объединение сетевых пачек в один фрейм со сжатием **Zstandard (zstd)**, снижающий число WebSocket-сообщений в 6–60 раз и существенно уменьшающий сетевой пинг.
  - Опциональный режим `--codec=legacy` для по-пакетного сжатия LZ4.
- **Сквозное шифрование (Zero-Knowledge):**
  - AES-256-GCM с деривацией ключа через scrypt. Промежуточные узлы не имеют доступа к заголовкам, URL, SNI и DNS.
  - Защита памяти и контроль размера пакетов (лимит фрейма LZ4 10 МБ).
- **Клиент Android (`android/`):**
  - Интерфейс Material 3 (светлая/тёмная тема).
  - Встроенный сканер QR-кодов.
  - Режимы работы: полный **VPN** (`VpnService` + `tun2socks`/gVisor) и локальный **SOCKS5-прокси**.
  - Раздельное туннелирование (Split Tunneling): белые и чёрные списки приложений.
  - Мониторинг скорости в реальном времени, пинг, время работы и статистика трафика.
  - Плитка быстрых настроек (Quick Settings Tile) в шторке Android.
- **Клиент Windows Desktop GUI (`desktop/`):**
  - Интерфейс на Wails v2 + Vue 3.
  - Режимы: Wintun (полноценный системный сетевой адаптер), системный прокси (SysProxy), SOCKS5 и встроенная Exit Node.
- **Серверная нода выхода (Linux Exit Node):**
  - Два режима работы: **Proxy** (userspace `net.Dial`, не требует root-прав) и **Raw Sockets** (высокоскоростной роутинг через сырые сокеты с отслеживанием портов).
  - Готовый `docker-compose.yml` и сценарии развёртывания в изолированном `network namespace` (защита от сбросов RST).
  - Поддержка systemd watchdog (`WatchdogSec=30s`).

---

## Скриншоты

### Windows Desktop (GUI)
<p align="center">
  <img src="docs/screenshots/desktop.jpg" width="85%" alt="OpenFlux Windows Desktop GUI" />
</p>

### Android-клиент
<table width="100%">
  <tr>
    <th width="25%" align="center">Главный экран</th>
    <th width="25%" align="center">Настройки</th>
    <th width="25%" align="center">Раздельный туннель</th>
    <th width="25%" align="center">Журнал событий</th>
  </tr>
  <tr>
    <td width="25%" align="center" valign="top"><img src="docs/screenshots/android_main.jpg" width="100%" alt="Главный экран" /></td>
    <td width="25%" align="center" valign="top"><img src="docs/screenshots/android_settings.jpg" width="100%" alt="Настройки" /></td>
    <td width="25%" align="center" valign="top"><img src="docs/screenshots/android_split_tunnel.jpg" width="100%" alt="Раздельный туннель" /></td>
    <td width="25%" align="center" valign="top"><img src="docs/screenshots/android_logs.jpg" width="100%" alt="Журнал событий" /></td>
  </tr>
</table>

---

## Быстрый старт

### 1. Сервер выхода (Linux VPS)

Сервер устанавливает исходящее соединение с облачной платформой. Открывать входящие порты в файрволе не требуется — нода работает за NAT.

#### Вариант A: Docker Compose (Рекомендуемый)

1. Клонируйте репозиторий и настройте переменные:
   ```bash
   git clone https://github.com/tatarinovs/OpenFlux.git
   cd OpenFlux
   cp deploy/openflux.env.example .env
   nano .env
   ```
2. Укажите ссылку на документ/комнату и пароль шифрования:
   ```env
   OPENFLUX_URL=https://disk.yandex.ru/i/ВАШ_ДОКУМЕНТ
   OPENFLUX_KEY=ваш_пароль_или_hex_ключ
   OPENFLUX_TRANSPORT=yandex
   ```
3. Запустите контейнер:
   ```bash
   docker compose up -d --build
   docker compose logs -f
   ```

#### Вариант Б: Прямой запуск бинарника (CLI)

```bash
# Режим Proxy (не требует root):
./openflux -exit-node -mode proxy -transport yandex -url "https://disk.yandex.ru/i/..." -key "ваш_пароль"

# Режим Raw Sockets (высокая производительность, требует root):
sudo iptables -I OUTPUT 1 -p tcp --tcp-flags RST RST -j DROP
sudo ./openflux -exit-node -mode raw -transport yandex -url "https://disk.yandex.ru/i/..." -key "ваш_пароль"
```

---

### 2. Подключение клиентов

1. **Android**:
   - Установите APK из каталога [releases/](releases/) или вкладки Releases на GitHub.
   - Вставьте ссылку на документ (или отсканируйте QR-код конфигурации с помощью встроенного сканера в настройках).
   - Введите пароль шифрования (тот же `OPENFLUX_KEY`, что и на сервере).
   - Выберите режим (VPN или Proxy) и нажмите кнопку подключения.
2. **Windows Desktop**:
   - Запустите `OpenFlux.exe` из папки [releases/](releases/).
   - Укажите транспорт, ссылку на документ и ключ шифрования.
   - Выберите желаемый режим (Wintun VPN, System Proxy или SOCKS5).
3. **CLI-клиент (Любая ОС)**:
   ```bash
   ./openflux -client -transport yandex -url "https://disk.yandex.ru/i/..." -key "ваш_пароль" -socks5 127.0.0.1:1080
   ```

---

## Параметры командной строки

| Флаг | По умолчанию | Описание |
|---|---|---|
| `-client` | `false` | Запуск в режиме клиента (SOCKS5 прокси) |
| `-exit-node` | `false` | Запуск в режиме выходной ноды |
| `-mode` | `proxy` | Режим выходной ноды: `proxy` (userspace, без root) или `raw` |
| `-transport` | `yandex` | Транспорт: `yandex`, `vyandex`, `cupsonline`, `oneme` |
| `-url` | `http://#` | Публичная ссылка на документ или комнату |
| `-key` | `""` | Пароль сквозного шифрования (или переменная `OPENFLUX_KEY`) |
| `-socks5` | `:1080` | Адрес прослушивания SOCKS5-сервера |
| `-ycookie` | `""` | Сессионные cookies Яндекса (при необходимости проверки сессии) |
| `-maxToken` | `""` | Токен авторизации MAX (для транспорта `oneme`) |
| `-maxUid` | `""` | ID пользователя MAX (для транспорта `oneme`) |
| `-debug` | `false` | Подробные отладочные логи |

---

## Сборка из исходников

- **Все платформы в один клик (Windows):**
  ```cmd
  build_all.bat
  ```
- **Android Release APK:**
  ```cmd
  build_release.bat
  ```
- **Windows Desktop (Wails):**
  ```cmd
  build_windows.bat
  ```
- **Linux сервер / CLI:**
  ```bash
  go build -trimpath -ldflags="-s -w" -o openflux .
  ```
