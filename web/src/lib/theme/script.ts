import { ACCENTS, DEFAULT_PREFERENCES, DENSITIES, MOTIONS, THEME_MODES } from "./theme";

// Runs in <head> before the first paint so a dark-mode user never sees a light flash.
// It must be self-contained (no imports at runtime), so it mirrors theme.ts:
// scopeForPath → cookieName → parsePreferences → themeAttributes. theme-script.test.ts
// checks that both produce the same attributes.
export function themeScript(): string {
  const allowed = JSON.stringify({
    mode: THEME_MODES,
    accent: ACCENTS,
    density: DENSITIES,
    motion: MOTIONS,
  });
  const defaults = JSON.stringify(DEFAULT_PREFERENCES);

  return `(function(){try{
var A=${allowed},D=${defaults},d=document.documentElement,p=location.pathname;
var s=(p==="/owner"||p.indexOf("/owner/")===0)?"owner":"learner",n="sat_"+s+"_appearance",raw="";
document.cookie.split(";").forEach(function(c){var i=c.indexOf("=");if(c.slice(0,i).trim()===n){try{raw=decodeURIComponent(c.slice(i+1))}catch(e){}}});
var v=raw.split("."),k=["mode","accent","density","motion"],r={};
k.forEach(function(x,i){r[x]=A[x].indexOf(v[i])>=0?v[i]:D[x]});
var dark=r.mode==="dark"||(r.mode==="system"&&window.matchMedia("(prefers-color-scheme: dark)").matches);
d.setAttribute("data-theme",dark?"dark":"light");d.setAttribute("data-accent",r.accent);
d.setAttribute("data-density",r.density);d.setAttribute("data-motion",r.motion);
}catch(e){}})();`;
}
