# Daedalus Web Admin — UI Redesign Design Spec

**Date:** 2026-04-13  
**Scope:** Global visual redesign — `web-admin/src`  
**Approach:** Enfoque 2 — Token override + Presentation layer rewrite of layout shell  
**Stack:** Angular 20 + CoreUI 5.5 + SCSS

---

## 1. Design Direction

**Aesthetic:** Industrial/technical — cyberpunk operational. Dense, dark, mission-critical. Evokes server room dashboards, DevOps terminals, and air-traffic control interfaces.

**Differentiator:** Two-accent brand identity derived from existing brand assets:
- Cian eléctrico `#00E5FF` — primary interactive accent (actions, active states, focus)
- Crimson `#CF2F4C` — brand anchor (logo, danger/critical semantic states)

These two colors are near-complementary on the color wheel, creating high visual tension that reads as "operational urgency" — appropriate for an orchestration system.

---

## 2. Color System

### Background Stack
```scss
--daedalus-bg-base:     #080C0F   // near-black with cold cyan tint — body background
--daedalus-bg-surface:  #0D1317   // cards, sidebar, panels
--daedalus-bg-elevated: #111A20   // dropdowns, modals, popovers
--daedalus-bg-overlay:  #162028   // hover states, selected table rows
```

### Accent Stack
```scss
--daedalus-primary:       #00E5FF              // primary interactive accent
--daedalus-primary-dim:   #00B8CC              // hover/secondary states
--daedalus-primary-glow:  rgba(0,229,255,0.15) // box-shadow glow
--daedalus-primary-trace: rgba(0,229,255,0.06) // background tints

--daedalus-brand:         #CF2F4C              // brand crimson (from logo)
--daedalus-brand-dim:     #A8253C              // brand hover
--daedalus-brand-glow:    rgba(207,47,76,0.20) // danger glow
```

### Text Stack
```scss
--daedalus-text-primary:   #E8F4F8   // near-white cold — main content
--daedalus-text-secondary: #7A9BA8   // labels, metadata, nav items default
--daedalus-text-muted:     #3D5A66   // placeholders, disabled, dividers
--daedalus-text-accent:    #00E5FF   // active values, highlighted data
```

### Semantic Status Colors
```scss
--daedalus-success: #00C896   // running, healthy
--daedalus-warning: #FFB300   // degraded, pending
--daedalus-danger:  #CF2F4C   // failed, critical — matches brand crimson
--daedalus-info:    #448AFF   // informational, syncing
```

### CoreUI Token Overrides (in `_theme.scss`)
Map CoreUI's `--cui-*` tokens to Daedalus tokens:
```scss
--cui-body-bg:            var(--daedalus-bg-base)
--cui-body-color:         var(--daedalus-text-primary)
--cui-primary:            var(--daedalus-primary)
--cui-primary-rgb:        0, 229, 255
--cui-link-color:         var(--daedalus-primary)
--cui-link-hover-color:   var(--daedalus-primary-dim)
--cui-border-color:       rgba(0,229,255,0.10)
--cui-card-bg:            var(--daedalus-bg-surface)
--cui-secondary-bg:       var(--daedalus-bg-elevated)
--cui-tertiary-bg:        var(--daedalus-bg-overlay)
```

---

## 3. Typography

### Fonts
- **Display / Data:** Geist Mono — loaded via CDN (`fonts.googleapis.com`)
- **Body / UI:** IBM Plex Sans — loaded via CDN

Both loaded with `font-display: swap` in `web-admin/src/index.html`.

### Usage
```scss
--daedalus-font-display: 'Geist Mono', monospace
--daedalus-font-body:    'IBM Plex Sans', system-ui, sans-serif

// Where each is used:
// Geist Mono   → sidebar brand text, table column headers, data values,
//                status badges, footer version, any numeric/code values
// IBM Plex Sans → nav item labels, card body text, form inputs, general UI copy
```

### Scale
```scss
--daedalus-text-xs:   0.6875rem   // 11px — table headers (monospace, uppercase)
--daedalus-text-sm:   0.8125rem   // 13px — badges, metadata
--daedalus-text-base: 0.9375rem   // 15px — body
--daedalus-text-lg:   1.0625rem   // 17px — card titles
--daedalus-text-xl:   1.25rem     // 20px — page headings
--daedalus-text-2xl:  1.625rem    // 26px — dashboard widget numbers
```

---

## 4. Layout Shell

### Files to modify
```
web-admin/src/app/layout/default-layout/
  default-layout.component.html   ← structural changes
  default-layout.component.scss   ← full rewrite
  default-layout/default-header/
    default-header.component.html ← strip demo data, clean markup
    default-header.component.scss ← rewrite
  default-layout/default-footer/
    default-footer.component.html ← simplify to brand + version
    default-footer.component.scss ← rewrite
```

### Sidebar
- Background: `--daedalus-bg-base`
- Right border: `1px solid rgba(0,229,255,0.12)` — cian halo
- **Scanline texture:** CSS `repeating-linear-gradient` of 1px semi-transparent lines (`rgba(0,0,0,0.15)`) every 2px — subtle CRT monitor effect applied as `::after` pseudo-element overlay on the sidebar (pointer-events: none, z-index above background but below content)
- **Brand area:** Daedalus signet in white + "NAGULA" brand text visible in the logo SVG renders in `#CF2F4C` naturally
- **Nav items default:** text `#7A9BA8`, IBM Plex Sans 14px, icon opacity 0.6
- **Nav item hover:** text `#E8F4F8`, background `rgba(0,229,255,0.06)`, transition `120ms ease`
- **Nav item active:** left border `3px solid #00E5FF` (animates from 0px width on activation), text `#00E5FF`, background `rgba(0,229,255,0.10)`, icon color `#00E5FF`

### Header
- Background: `--daedalus-bg-surface`
- Bottom border: `1px solid rgba(0,229,255,0.08)`
- Breadcrumb: separators in `--daedalus-primary-dim`, items in muted, current page in `--daedalus-text-primary`
- User dropdown: avatar with `2px solid rgba(0,229,255,0.40)` border
- Sidebar toggler hover: icon color transitions to `#00E5FF`
- Remove: demo notification/message dropdowns from the component (not wired, just visual noise)

### Footer
- Background: `--daedalus-bg-base`
- Top border: `1px solid rgba(0,229,255,0.06)`
- Left: `Daedalus Orchestrator` in `--daedalus-text-muted`, IBM Plex Sans
- Right: `v0.1.0` in `#00E5FF`, Geist Mono — hardcoded string constant in `default-footer.component.ts` as `readonly version = 'v0.1.0'`
- Copyright center: `© 2026` in `--daedalus-text-muted`

---

## 5. Component Overrides (`_custom.scss`)

### Cards
```scss
.card {
  background: var(--daedalus-bg-surface);
  border: 1px solid rgba(0,229,255,0.10);
  border-radius: 2px;  // near-zero — industrial, not bubbly

  .card-header {
    background: var(--daedalus-bg-elevated);
    border-bottom: 1px solid rgba(0,229,255,0.08);
    font-family: var(--daedalus-font-display);
    font-size: var(--daedalus-text-sm);
    letter-spacing: 0.05em;
    text-transform: uppercase;
    color: var(--daedalus-text-secondary);
  }

  &:hover {
    border-color: rgba(0,229,255,0.30);
    box-shadow: 0 0 20px rgba(0,229,255,0.05);
    transition: border-color 200ms ease, box-shadow 200ms ease;
  }
}
```

### Tables
```scss
.table {
  th {
    font-family: var(--daedalus-font-display);
    font-size: var(--daedalus-text-xs);
    letter-spacing: 0.1em;
    text-transform: uppercase;
    color: var(--daedalus-text-secondary);
    border-bottom: 1px solid rgba(0,229,255,0.15);
  }

  tr:nth-child(even) {
    background: rgba(0,229,255,0.02);
  }

  tr:hover {
    background: rgba(0,229,255,0.06);
    transition: background 120ms ease;
    cursor: pointer;
  }
}
```

### Buttons
```scss
.btn-primary {
  background: transparent;
  border: 1px solid var(--daedalus-primary);
  color: var(--daedalus-primary);
  border-radius: 2px;
  font-family: var(--daedalus-font-body);

  &:hover { background: rgba(0,229,255,0.12); }
  &:active { background: rgba(0,229,255,0.20); }
}

.btn-danger {
  background: transparent;
  border: 1px solid var(--daedalus-brand);
  color: var(--daedalus-brand);

  &:hover { background: rgba(207,47,76,0.12); }
}
```

### Form Inputs
```scss
.form-control {
  background: var(--daedalus-bg-surface);
  border: 1px solid var(--daedalus-text-muted);
  color: var(--daedalus-text-primary);
  border-radius: 2px;

  &::placeholder { color: var(--daedalus-text-muted); }

  &:focus {
    border-color: var(--daedalus-primary);
    box-shadow: 0 0 0 3px rgba(0,229,255,0.10);
  }
}
```

### Status Badges
```scss
.badge-running  { background: rgba(0,200,150,0.15); color: #00C896; border: 1px solid rgba(0,200,150,0.30); }
.badge-failed   { background: rgba(207,47,76,0.15);  color: #CF2F4C; border: 1px solid rgba(207,47,76,0.30); }
.badge-pending  { background: rgba(255,179,0,0.15);  color: #FFB300; border: 1px solid rgba(255,179,0,0.30); }
.badge-healthy  { background: rgba(0,200,150,0.15);  color: #00C896; border: 1px solid rgba(0,200,150,0.30); }
.badge-info     { background: rgba(68,138,255,0.15); color: #448AFF; border: 1px solid rgba(68,138,255,0.30); }
```

---

## 6. Motion

All animations are CSS-only. `@media (prefers-reduced-motion: reduce)` disables all transitions.

### 1. Sidebar nav active indicator
```scss
.nav-link::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  height: 100%;
  width: 0;
  background: var(--daedalus-primary);
  transition: width 80ms ease;
}
.nav-link.active::before { width: 3px; }
```

### 2. Card hover glow
```scss
.card { transition: border-color 200ms ease, box-shadow 200ms ease; }
```

### 3. Dashboard page load stagger
```scss
@keyframes fadeSlideUp {
  from { opacity: 0; transform: translateY(8px); }
  to   { opacity: 1; transform: translateY(0); }
}

.dashboard-widget {
  animation: fadeSlideUp 300ms ease both;

  &:nth-child(1) { animation-delay: 0ms; }
  &:nth-child(2) { animation-delay: 60ms; }
  &:nth-child(3) { animation-delay: 120ms; }
  &:nth-child(4) { animation-delay: 180ms; }
}
```

---

## 7. Files Changed Summary

| File | Action |
|------|--------|
| `web-admin/src/index.html` | Add Google Fonts link tags (Geist Mono + IBM Plex Sans) |
| `web-admin/src/scss/_theme.scss` | Full rewrite with Daedalus tokens + CoreUI overrides |
| `web-admin/src/scss/_custom.scss` | Component overrides: cards, tables, buttons, inputs, badges |
| `web-admin/src/scss/styles.scss` | No structural changes — already imports `theme` and `custom` |
| `web-admin/src/app/layout/default-layout/default-layout.component.scss` | Rewrite sidebar styles |
| `web-admin/src/app/layout/default-layout/default-layout.component.html` | Minor structural: scanline pseudo-element wrapper |
| `web-admin/src/app/layout/default-layout/default-header/default-header.component.html` | Strip demo notification/message data |
| `web-admin/src/app/layout/default-layout/default-header/default-header.component.scss` | Rewrite header styles |
| `web-admin/src/app/layout/default-layout/default-footer/default-footer.component.html` | Simplify to brand + version + copyright |
| `web-admin/src/app/layout/default-layout/default-footer/default-footer.component.scss` | Rewrite footer styles |

---

## 8. Out of Scope

- Rewriting product view components (dashboard, tenants, cluster, etc.) — shell only
- Removing CoreUI demo routes (`base/`, `buttons/`, `forms/`, etc.)
- Adding Node Schedulers to sidebar nav
- Backend integration changes
- Unit test updates

These are natural follow-up tasks after the visual shell is validated.

---

## 9. Accessibility

- All color combinations meet WCAG AA (4.5:1) for text on background
- `prefers-reduced-motion` support for all animations
- Focus states use visible `box-shadow` outlines, not just color changes
- Sidebar active state uses border + color (not color alone) for contrast
