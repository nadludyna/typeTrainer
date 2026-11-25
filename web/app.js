// Entry point of the frontend.

import { t } from "./i18n.js";
import { FX } from "./fx.js";
import { el, S, store, isTouch } from "./state.js";
import { getJSON } from "./api.js";
import { labelTips } from "./tools.js";
import { autoKb } from "./keyboard.js";
import { showIntro } from "./intro.js";
import { renderProfile, loadAccount } from "./profile.js";
import { loadLanguage } from "./languages.js";
import { loadSettings } from "./settings.js";

import { startView } from "./input.js";

// ---------- start ----------
(async function init() {
  renderProfile();
  loadSettings();
  autoKb();
  labelTips();
  if (isTouch()) el.textBox.classList.add("blurred");
  try {
    await loadAccount();
    S.languages = await getJSON("/languages", true);
    await loadLanguage(S.languages[0].id);
  } catch (err) {
    console.error(err);
    el.text.textContent = t("err.server");
  }
  startView();
  const flash = store.get("tt.flash");
  if (flash) {
    store.set("tt.flash", "");
    FX.toast(t("toast.imported", { n: flash }));
  }
  if (!store.get("tt.introDone") && S.layout && !isTouch()) showIntro();
})();
