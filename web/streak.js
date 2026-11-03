// The clean-word streak: a number of the top line that pops, and changes colour at each milestone.

import { el, S, reducedMotion } from "./state.js";

const STREAK_TIERS = [5, 10, 15, 25, 35, 50, 75, 100];
const streakTier = (n) => (n ? 1 + STREAK_TIERS.filter((x) => n >= x).length : 0);
const streakOn = () => S.settings.streak && S.mode === "freetype";

let streak = 0;

export function cleanWord() {
  streak++;
  renderStreak();
  if (!streakOn() || reducedMotion()) return;
  const milestone = STREAK_TIERS.includes(streak);
  el.streakN.animate([{ transform: `scale(${milestone ? 1.8 : 1.3})` }, { transform: "none" }], { duration: milestone ? 420 : 220, easing: "cubic-bezier(.2,.9,.3,1.3)" });
}
export function breakStreak() {
  const was = streak;
  streak = 0;
  renderStreak();
  if (!was || !streakOn() || reducedMotion()) return;
  el.streak.animate([{ transform: "none" }, { transform: "translateX(-3px)" }, { transform: "translateX(3px)" }, { transform: "translateX(-2px)" }, { transform: "none" }], { duration: 260, easing: "ease-out" });
}
function renderStreak() {
  el.streakN.textContent = streak;
  el.streak.dataset.tier = streakTier(streak);
}
