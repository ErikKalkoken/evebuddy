// Mobile menu toggle.
const menuToggle = document.getElementById("menuToggle");
if (menuToggle) {
  menuToggle.addEventListener("click", () => {
    const open = document.getElementById("navLinks").classList.toggle("open");
    menuToggle.setAttribute("aria-expanded", open);
  });
}

// Returns the os key used in data/platforms.yaml, or "" when unknown.
function detectOS() {
  const ua = navigator.userAgent;
  const p = (navigator.userAgentData && navigator.userAgentData.platform) || navigator.platform || "";
  if (/android/i.test(ua)) return "android"; // before linux: Android UAs contain "Linux"
  if (/iphone|ipad|ipod/i.test(ua)) return "";
  if (/win/i.test(p) || /windows/i.test(ua)) return "windows";
  if (/mac/i.test(p) || /mac os x/i.test(ua)) return ""; // Apple Silicon, Intel and iPads look alike, so no pick
  if (/linux|x11/i.test(p) || /linux/i.test(ua)) return "linux";
  return "";
}

const os = detectOS();

// Hero: point the main button to the file for this platform.
const hero = document.getElementById("heroDownload");
if (hero && os) {
  const data = JSON.parse(hero.dataset.downloads);
  const d = data.downloads.find((x) => x.os === os);
  if (d) {
    hero.href = d.url;
    hero.querySelector(".os-icon").innerHTML = d.icon;
    document.getElementById("heroLabel").textContent = hero.dataset.labelPlatform.replace("{platform}", d.title);
    const meta = [data.version && `Version ${data.version}`, d.size].filter(Boolean).join(" · ");
    if (meta) document.getElementById("heroMeta").textContent = meta;
  }
}

// Page TOC: highlight the entry of the section being read.
const targets = [...document.querySelectorAll(".toc a[href^='#']")]
  .map((a) => [a, document.getElementById(decodeURIComponent(a.hash.slice(1)))])
  .filter(([, h]) => h);
if (targets.length) {
  const offset = 100; // just below the sticky header (headings use scroll-margin-top: 80px)
  let current = null;
  let pending = false;
  const update = () => {
    pending = false;
    let active = null;
    const atBottom =
      window.scrollY > 0 && window.innerHeight + window.scrollY >= document.documentElement.scrollHeight - 2;
    if (atBottom) {
      active = targets[targets.length - 1][0]; // a short last section never reaches the top
    } else {
      for (const [a, h] of targets) {
        if (h.getBoundingClientRect().top > offset) break;
        active = a;
      }
    }
    if (active === current) return;
    current?.classList.remove("active");
    active?.classList.add("active");
    current = active;
    // Keep the active entry visible when the TOC itself scrolls (scrollIntoView would move the page too).
    const toc = active?.closest(".toc");
    if (toc && toc.scrollHeight > toc.clientHeight) {
      const a = active.getBoundingClientRect();
      const t = toc.getBoundingClientRect();
      const margin = 24;
      if (a.top < t.top + margin) toc.scrollTop -= t.top + margin - a.top;
      else if (a.bottom > t.bottom - margin) toc.scrollTop += a.bottom - (t.bottom - margin);
    }
  };
  const schedule = () => {
    if (pending) return;
    pending = true;
    requestAnimationFrame(update);
  };
  window.addEventListener("scroll", schedule, { passive: true });
  window.addEventListener("resize", schedule);
  update();
}

// Download page: highlight the card for this platform.
const grid = document.getElementById("dlGrid");
if (grid && os) {
  const card = grid.querySelector(`[data-os="${os}"]`);
  if (card) {
    card.classList.add("recommended");
    card.querySelector(".btn").classList.add("btn-primary");
  }
}
