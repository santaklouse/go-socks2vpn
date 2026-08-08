const translations = {
  en: {
    skip: "Skip to content",
    "nav.features": "Features", "nav.how": "How it works", "nav.install": "Install", "nav.download": "Download",
    "hero.eyebrow": "Source available · Native · Cross-platform",
    "hero.title": "Your SOCKS proxy.<br><span>Now system-wide.</span>",
    "hero.lead": "Turn a SOCKS4 or SOCKS5 server into a full-device VPN with one lightweight Go client. No external tun2socks process. No browser extensions.",
    "hero.install": "Install go-socks2vpn", "hero.source": "View source",
    "preview.kicker": "SOCKS → SYSTEM VPN", "preview.title": "Connection", "preview.connected": "Connected", "preview.protocol": "Protocol", "preview.server": "Server", "preview.port": "Port", "preview.disconnect": "Disconnect", "preview.session": "Session", "preview.downloaded": "Downloaded", "preview.uploaded": "Uploaded",
    "chip.engine": "Embedded engine", "chip.process": "No extra process", "chip.rollback": "Safe rollback", "chip.routes": "Routes restored",
    "trust.platforms": "platforms", "trust.protocols": "SOCKS protocols", "trust.processes": "external processes", "trust.opensource": "public source",
    "features.eyebrow": "One tunnel. Every app.", "features.title": "Built for the traffic<br><span>your browser cannot reach.</span>", "features.lead": "Route applications that do not understand SOCKS through the same proxy—without configuring each one separately.",
    "features.system.title": "System-wide routing", "features.system.text": "Desktop routes are configured automatically and restored when the tunnel stops. Android uses the native, rootless VpnService.",
    "features.engine.title": "Embedded Go engine", "features.engine.text": "tun2socks is compiled directly into the client as a Go dependency. Nothing extra to download or supervise.",
    "features.stats.title": "Live traffic stats", "features.stats.text": "See session totals and current download/upload speed in the GUI or enable them in the CLI.",
    "features.dns.title": "Browser-ready DNS", "features.dns.text": "Android sends DNS-over-HTTPS through SOCKS and forces reliable TCP fallback when QUIC is unavailable.",
    "features.links.title": "One-tap configuration", "features.links.text": "Open a socks2vpn:// link to fill the GUI safely. The application never connects automatically.",
    "features.verify.title": "Verified releases", "features.verify.text": "Every release includes SHA-256 checksums. Windows verifies Wintun before loading it.",
    "how.eyebrow": "Simple by design", "how.title": "From proxy to full tunnel in three steps.",
    "how.one.title": "Enter your proxy", "how.one.text": "Choose SOCKS4 or SOCKS5 and provide the server details. Credentials stay in process memory.",
    "how.two.title": "Create the tunnel", "how.two.text": "The embedded engine opens a TUN interface and keeps the proxy connection outside its own route.",
    "how.three.title": "Use any application", "how.three.text": "Your TCP traffic follows the proxy. SOCKS5 can also carry UDP when the server supports it.",
    "install.eyebrow": "Ready when you are", "install.title": "Install in under a minute.", "install.lead": "Choose your platform. Releases are built automatically and published with checksums.", "install.command": "TERMINAL", "install.copy": "Copy", "install.copied": "Copied", "install.release": "Latest release", "install.docs": "Read installation docs →",
    "install.unix.title": "Install the desktop GUI", "install.unix.text": "Detects your OS and CPU, verifies SHA-256, and registers configuration links.", "install.unix.note": "Launch with administrator privileges so the app can create routes. The GUI will alert you if privileges are missing.",
    "install.windows.title": "Install from PowerShell", "install.windows.text": "Installs the GUI, adds it to PATH, creates a Start menu shortcut, and registers socks2vpn:// links.", "install.windows.note": "Open PowerShell normally. Windows will show the standard UAC prompt when the elevated app starts.",
    "install.android.title": "Install the Android APK", "install.android.text": "Download the signed release APK, then install it on your device or through ADB.", "install.android.note": "Android uses the system VpnService and does not require root access.",
    "install.cli.title": "Install the Go CLI", "install.cli.text": "Build and install the latest command-line client directly with Go.", "install.cli.note": "Run the resulting binary with sudo or Administrator privileges to create system routes.",
    "install.docker.title": "Run the container", "install.docker.text": "Starts the published multi-architecture image with the TUN device and required capability.", "install.docker.note": "Routes change inside the container network namespace. Use the Compose sidecar setup for other containers.",
    "security.eyebrow": "Transparent and inspectable", "security.title": "Your tunnel should not be a black box.", "security.text": "The source is public, the client uses an embedded Go network stack, keeps passwords out of preferences, verifies native downloads, and restores network settings after a normal shutdown.", "security.link": "Inspect the source on GitHub →",
    "security.one.title": "No proxy password storage", "security.one.text": "Secrets remain in memory only", "security.two.title": "Automatic route rollback", "security.two.text": "Clean restoration on shutdown", "security.three.title": "Reproducible release pipeline", "security.three.text": "Multi-platform CI and checksums",
    "cta.title": "Make your SOCKS proxy useful everywhere.", "cta.text": "Download the latest release and connect your whole device in minutes.", "cta.button": "Get go-socks2vpn",
    "footer.made": "Built in Go. Source available on GitHub.", "footer.releases": "Releases", "footer.issues": "Issues"
  },
  ru: {
    skip: "Перейти к содержимому",
    "nav.features": "Возможности", "nav.how": "Как работает", "nav.install": "Установка", "nav.download": "Скачать",
    "hero.eyebrow": "Исходный код доступен · Нативно · Кроссплатформенно",
    "hero.title": "Ваш SOCKS-прокси.<br><span>Теперь для всей системы.</span>",
    "hero.lead": "Превратите SOCKS4- или SOCKS5-сервер в VPN для всего устройства с помощью лёгкого Go-клиента. Без внешнего процесса tun2socks и расширений браузера.",
    "hero.install": "Установить go-socks2vpn", "hero.source": "Исходный код",
    "preview.kicker": "SOCKS → СИСТЕМНЫЙ VPN", "preview.title": "Подключение", "preview.connected": "Подключено", "preview.protocol": "Протокол", "preview.server": "Сервер", "preview.port": "Порт", "preview.disconnect": "Отключить", "preview.session": "Сеанс", "preview.downloaded": "Получено", "preview.uploaded": "Отправлено",
    "chip.engine": "Встроенный движок", "chip.process": "Без отдельного процесса", "chip.rollback": "Безопасный откат", "chip.routes": "Маршруты восстановлены",
    "trust.platforms": "платформ", "trust.protocols": "SOCKS-протокола", "trust.processes": "внешних процессов", "trust.opensource": "публичный код",
    "features.eyebrow": "Один туннель. Все приложения.", "features.title": "Для трафика,<br><span>которому мало браузера.</span>", "features.lead": "Направляйте через один прокси приложения, которые не умеют работать с SOCKS, без отдельной настройки каждого из них.",
    "features.system.title": "Системная маршрутизация", "features.system.text": "Маршруты рабочего стола настраиваются автоматически и восстанавливаются после остановки. Android использует системный VpnService без root.",
    "features.engine.title": "Встроенный Go-движок", "features.engine.text": "tun2socks компилируется прямо в клиент как Go-зависимость. Нечего отдельно скачивать или контролировать.",
    "features.stats.title": "Живая статистика", "features.stats.text": "Смотрите общий трафик сеанса и текущие скорости в GUI либо включите статистику в CLI.",
    "features.dns.title": "DNS для браузеров", "features.dns.text": "Android отправляет DNS-over-HTTPS через SOCKS и надёжно переключает QUIC на TCP, если UDP недоступен.",
    "features.links.title": "Настройка одним нажатием", "features.links.text": "Откройте ссылку socks2vpn://, чтобы безопасно заполнить GUI. Программа никогда не подключается автоматически.",
    "features.verify.title": "Проверяемые релизы", "features.verify.text": "Каждый релиз содержит SHA-256. Windows проверяет Wintun до его загрузки.",
    "how.eyebrow": "Простая архитектура", "how.title": "От прокси до полного туннеля за три шага.",
    "how.one.title": "Укажите прокси", "how.one.text": "Выберите SOCKS4 или SOCKS5 и введите адрес сервера. Учётные данные остаются только в памяти процесса.",
    "how.two.title": "Создайте туннель", "how.two.text": "Встроенный движок открывает TUN-интерфейс и оставляет соединение с прокси за пределами собственного маршрута.",
    "how.three.title": "Используйте любое приложение", "how.three.text": "TCP-трафик проходит через прокси. SOCKS5 также передаёт UDP, если сервер это поддерживает.",
    "install.eyebrow": "Можно начинать", "install.title": "Установка меньше чем за минуту.", "install.lead": "Выберите платформу. Релизы собираются автоматически и публикуются с контрольными суммами.", "install.command": "ТЕРМИНАЛ", "install.copy": "Копировать", "install.copied": "Скопировано", "install.release": "Последний релиз", "install.docs": "Документация по установке →",
    "install.unix.title": "Установка Desktop GUI", "install.unix.text": "Определяет ОС и процессор, проверяет SHA-256 и регистрирует ссылки конфигурации.", "install.unix.note": "Запускайте с правами администратора для создания маршрутов. GUI предупредит, если прав недостаточно.",
    "install.windows.title": "Установка из PowerShell", "install.windows.text": "Устанавливает GUI, добавляет его в PATH, создаёт ярлык в меню «Пуск» и регистрирует ссылки socks2vpn://.", "install.windows.note": "Откройте обычный PowerShell. При запуске приложения Windows покажет стандартный запрос UAC.",
    "install.android.title": "Установка Android APK", "install.android.text": "Скачайте подписанный APK релиза и установите его на устройство или через ADB.", "install.android.note": "Android использует системный VpnService и не требует root-доступа.",
    "install.cli.title": "Установка Go CLI", "install.cli.text": "Соберите и установите последнюю консольную версию напрямую через Go.", "install.cli.note": "Для создания системных маршрутов запускайте бинарник с sudo или правами администратора.",
    "install.docker.title": "Запуск контейнера", "install.docker.text": "Запускает опубликованный multi-arch образ с TUN-устройством и необходимой capability.", "install.docker.note": "Маршруты меняются внутри network namespace контейнера. Для других контейнеров используйте sidecar из Compose.",
    "security.eyebrow": "Прозрачно и проверяемо", "security.title": "Туннель не должен быть чёрным ящиком.", "security.text": "Исходный код клиента опубликован, сетевой стек Go встроен, пароли не сохраняются в настройках, нативные загрузки проверяются, а сеть восстанавливается после штатного завершения.", "security.link": "Изучить код на GitHub →",
    "security.one.title": "Без хранения пароля прокси", "security.one.text": "Секреты остаются только в памяти", "security.two.title": "Автоматический откат маршрутов", "security.two.text": "Чистое восстановление при выходе", "security.three.title": "Воспроизводимый выпуск", "security.three.text": "Multi-platform CI и контрольные суммы",
    "cta.title": "Используйте SOCKS-прокси во всей системе.", "cta.text": "Скачайте последний релиз и подключите всё устройство за несколько минут.", "cta.button": "Скачать go-socks2vpn",
    "footer.made": "Создано на Go. Исходный код доступен на GitHub.", "footer.releases": "Релизы", "footer.issues": "Задачи"
  }
};

const installOptions = {
  unix: {
    title: "install.unix.title", description: "install.unix.text", note: "install.unix.note",
    code: "curl -fsSL https://raw.githubusercontent.com/santaklouse/go-socks2vpn/main/scripts/install-gui.sh | sh",
    link: "https://github.com/santaklouse/go-socks2vpn/releases/latest"
  },
  windows: {
    title: "install.windows.title", description: "install.windows.text", note: "install.windows.note",
    code: "irm https://raw.githubusercontent.com/santaklouse/go-socks2vpn/main/scripts/install-gui.ps1 | iex",
    link: "https://github.com/santaklouse/go-socks2vpn/releases/latest"
  },
  android: {
    title: "install.android.title", description: "install.android.text", note: "install.android.note",
    code: "adb install -r go-socks2vpn-android.apk",
    link: "https://github.com/santaklouse/go-socks2vpn/releases/latest/download/go-socks2vpn-android.apk"
  },
  cli: {
    title: "install.cli.title", description: "install.cli.text", note: "install.cli.note",
    code: "go install github.com/santaklouse/go-socks2vpn/cmd/socks2vpn@latest",
    link: "https://github.com/santaklouse/go-socks2vpn/releases/latest"
  },
  docker: {
    title: "install.docker.title", description: "install.docker.text", note: "install.docker.note",
    code: "docker run --rm --init --name socks2vpn --cap-add=NET_ADMIN --device=/dev/net/tun:/dev/net/tun ghcr.io/santaklouse/go-socks2vpn:latest --proxy='socks4://192.168.192.100:9050' --interface=eth0",
    link: "https://github.com/santaklouse/go-socks2vpn/pkgs/container/go-socks2vpn"
  }
};

let language = localStorage.getItem("go-socks2vpn-language") || (navigator.language.toLowerCase().startsWith("ru") ? "ru" : "en");
let activeInstall = "unix";

function t(key) {
  return translations[language][key] || translations.en[key] || key;
}

function applyLanguage(nextLanguage) {
  language = nextLanguage === "ru" ? "ru" : "en";
  document.documentElement.lang = language;
  localStorage.setItem("go-socks2vpn-language", language);
  document.querySelectorAll("[data-i18n]").forEach((element) => {
    element.textContent = t(element.dataset.i18n);
  });
  document.querySelectorAll("[data-i18n-html]").forEach((element) => {
    element.innerHTML = t(element.dataset.i18nHtml);
  });
  renderInstall(activeInstall);
}

function renderInstall(name) {
  const option = installOptions[name];
  if (!option) return;
  activeInstall = name;
  document.getElementById("install-title").textContent = t(option.title);
  document.getElementById("install-description").textContent = t(option.description);
  document.getElementById("install-note").textContent = t(option.note);
  document.getElementById("install-code").textContent = option.code;
  document.getElementById("install-primary-link").href = option.link;
  document.querySelectorAll("[data-tab]").forEach((button) => {
    const selected = button.dataset.tab === name;
    button.setAttribute("aria-selected", String(selected));
    button.tabIndex = selected ? 0 : -1;
  });
  document.getElementById("install-panel").setAttribute("aria-labelledby", `tab-${name}`);
}

document.querySelector(".language-toggle").addEventListener("click", () => applyLanguage(language === "en" ? "ru" : "en"));
const installTabs = [...document.querySelectorAll("[data-tab]")];
installTabs.forEach((button, index) => {
  button.addEventListener("click", () => renderInstall(button.dataset.tab));
  button.addEventListener("keydown", (event) => {
    let nextIndex;
    if (event.key === "ArrowRight") nextIndex = (index + 1) % installTabs.length;
    if (event.key === "ArrowLeft") nextIndex = (index - 1 + installTabs.length) % installTabs.length;
    if (event.key === "Home") nextIndex = 0;
    if (event.key === "End") nextIndex = installTabs.length - 1;
    if (nextIndex === undefined) return;
    event.preventDefault();
    installTabs[nextIndex].focus();
    renderInstall(installTabs[nextIndex].dataset.tab);
  });
});
document.querySelector(".copy-button").addEventListener("click", async (event) => {
  const button = event.currentTarget;
  try {
    await navigator.clipboard.writeText(document.getElementById("install-code").textContent);
    button.textContent = t("install.copied");
    window.setTimeout(() => { button.textContent = t("install.copy"); }, 1600);
  } catch {
    const selection = window.getSelection();
    const range = document.createRange();
    range.selectNodeContents(document.getElementById("install-code"));
    selection.removeAllRanges();
    selection.addRange(range);
  }
});

const observer = "IntersectionObserver" in window
  ? new IntersectionObserver((entries) => entries.forEach((entry) => {
      if (entry.isIntersecting) {
        entry.target.classList.add("is-visible");
        observer.unobserve(entry.target);
      }
    }), { threshold: 0.12 })
  : null;

document.querySelectorAll(".reveal").forEach((element) => {
  if (observer) observer.observe(element);
  else element.classList.add("is-visible");
});

document.getElementById("year").textContent = new Date().getFullYear();
applyLanguage(language);
