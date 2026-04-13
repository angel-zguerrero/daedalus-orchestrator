# Daedalus UI Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Apply a distinctive industrial/technical visual identity to the Daedalus Web Admin shell — dark background, cian electric accent, brand crimson, Geist Mono + IBM Plex Sans typography.

**Architecture:** Override CoreUI CSS custom properties (`--cui-*`) with Daedalus design tokens defined in `_theme.scss`. Add component-level visual overrides in `_custom.scss`. Rewrite the layout shell components (sidebar, header, footer) for a precise, opinionated visual result.

**Tech Stack:** Angular 20, CoreUI 5.5, SCSS, Nx (`npx nx build daedalus-web-admin`)

---

## Context

All commands are run from the workspace root: `/Users/angel/Documents/daedalus-orchestrator-project/daedalus-orchestrator`

Build command: `npx nx build daedalus-web-admin`
Serve command: `npx nx serve daedalus-web-admin`

Key files at a glance:
```
web-admin/src/
  index.html                                          ← add Google Fonts
  scss/
    styles.scss                                       ← already imports theme + custom, no changes needed
    _theme.scss                                       ← rewrite: design tokens + CoreUI overrides
    _custom.scss                                      ← add: component overrides (cards, tables, buttons, inputs, badges)
  app/layout/default-layout/
    default-layout.component.html                     ← add scanline wrapper div
    default-layout.component.scss                     ← rewrite: sidebar + scrollbar styles
    default-header/
      default-header.component.ts                     ← remove unused demo data properties
      default-header.component.html                   ← already clean, minor class additions
      default-header.component.scss                   ← write: header styles
    default-footer/
      default-footer.component.ts                     ← add version constant
      default-footer.component.html                   ← rewrite: brand + version + copyright
      default-footer.component.scss                   ← write: footer styles
```

---

## Task 1: Google Fonts + Daedalus Design Tokens

**Files:**
- Modify: `web-admin/src/index.html`
- Modify: `web-admin/src/scss/_theme.scss`

### Step 1.1: Add Google Fonts to index.html

Replace the contents of `web-admin/src/index.html` with:

```html
<!DOCTYPE html>
<!--
* CoreUI - Angular 20 Free Admin Template
* @version v5.5.3
* @link https://coreui.io/angular/
* Copyright (c) 2017-2025 creativeLabs Łukasz Holeczek
* License: MIT
-->
<html lang="en">
<head>
  <meta charset="utf-8">
  <base href="./">
  <meta content="width=device-width, initial-scale=1, shrink-to-fit=no" name="viewport" />
  <meta content="Daedalus Orchestrator" name="description" />
  <meta content="Daedalus" name="author" />
  <meta
    content="Daedalus,Daedalus orchestrator,event driven application,queues,state machine"
    name="keyword"
  />
  <link href="assets/favicon.ico" rel="icon" type="image/x-icon">
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Geist+Mono:wght@300;400;500;600&family=IBM+Plex+Sans:wght@300;400;500;600&display=swap" rel="stylesheet">
  <title>Daedalus</title>
</head>
<body>
<noscript>You need to enable JavaScript to run this app.</noscript>
<app-root>
  <div class="fade show" style="text-align: center; padding-top: calc(100vh / 2); height: 100vh">
    <i class="spinner-grow spinner-grow-sm"></i>
    <span class="m-1">Loading...</span>
  </div>
</app-root>
</body>
</html>
```

### Step 1.2: Rewrite `_theme.scss` with Daedalus design tokens + CoreUI overrides

Replace the entire contents of `web-admin/src/scss/_theme.scss` with:

```scss
@use "@coreui/coreui/scss/mixins/transition" as *;
@use "@coreui/coreui/scss/mixins/color-mode" as *;

// ─── Daedalus Design Tokens ──────────────────────────────────────────────────

:root,
[data-coreui-theme="dark"] {
  // Background stack
  --daedalus-bg-base:      #080C0F;
  --daedalus-bg-surface:   #0D1317;
  --daedalus-bg-elevated:  #111A20;
  --daedalus-bg-overlay:   #162028;

  // Primary accent — cian eléctrico
  --daedalus-primary:        #00E5FF;
  --daedalus-primary-dim:    #00B8CC;
  --daedalus-primary-glow:   rgba(0, 229, 255, 0.15);
  --daedalus-primary-trace:  rgba(0, 229, 255, 0.06);
  --daedalus-primary-border: rgba(0, 229, 255, 0.12);

  // Brand accent — crimson del logo
  --daedalus-brand:      #CF2F4C;
  --daedalus-brand-dim:  #A8253C;
  --daedalus-brand-glow: rgba(207, 47, 76, 0.20);

  // Text stack
  --daedalus-text-primary:   #E8F4F8;
  --daedalus-text-secondary: #7A9BA8;
  --daedalus-text-muted:     #3D5A66;
  --daedalus-text-accent:    #00E5FF;

  // Semantic status colors
  --daedalus-success: #00C896;
  --daedalus-warning: #FFB300;
  --daedalus-danger:  #CF2F4C;
  --daedalus-info:    #448AFF;

  // Typography
  --daedalus-font-display: 'Geist Mono', monospace;
  --daedalus-font-body:    'IBM Plex Sans', system-ui, sans-serif;

  // Type scale
  --daedalus-text-xs:   0.6875rem;  // 11px — table headers
  --daedalus-text-sm:   0.8125rem;  // 13px — badges, metadata
  --daedalus-text-base: 0.9375rem;  // 15px — body
  --daedalus-text-lg:   1.0625rem;  // 17px — card titles
  --daedalus-text-xl:   1.25rem;    // 20px — page headings
  --daedalus-text-2xl:  1.625rem;   // 26px — dashboard numbers

  // ─── CoreUI token overrides ───────────────────────────────────────────────
  --cui-body-bg:             var(--daedalus-bg-base);
  --cui-body-color:          var(--daedalus-text-primary);
  --cui-secondary-color:     var(--daedalus-text-secondary);
  --cui-tertiary-color:      var(--daedalus-text-muted);
  --cui-tertiary-bg:         var(--daedalus-bg-base);
  --cui-secondary-bg:        var(--daedalus-bg-elevated);

  --cui-primary:             var(--daedalus-primary);
  --cui-primary-rgb:         0, 229, 255;
  --cui-link-color:          var(--daedalus-primary);
  --cui-link-hover-color:    var(--daedalus-primary-dim);

  --cui-border-color:        rgba(0, 229, 255, 0.10);
  --cui-border-color-translucent: rgba(0, 229, 255, 0.08);

  --cui-card-bg:             var(--daedalus-bg-surface);
  --cui-card-border-color:   rgba(0, 229, 255, 0.10);
  --cui-card-cap-bg:         var(--daedalus-bg-elevated);

  --cui-dark-bg-subtle:      var(--daedalus-bg-base);

  --cui-success:             var(--daedalus-success);
  --cui-success-rgb:         0, 200, 150;
  --cui-warning:             var(--daedalus-warning);
  --cui-warning-rgb:         255, 179, 0;
  --cui-danger:              var(--daedalus-danger);
  --cui-danger-rgb:          207, 47, 76;
  --cui-info:                var(--daedalus-info);
  --cui-info-rgb:            68, 138, 255;

  --cui-font-sans-serif:     var(--daedalus-font-body);
  --cui-font-monospace:      var(--daedalus-font-display);

  --cui-sidebar-bg:          var(--daedalus-bg-base);
  --cui-sidebar-nav-link-color: var(--daedalus-text-secondary);
  --cui-sidebar-nav-link-hover-color: var(--daedalus-text-primary);
  --cui-sidebar-nav-link-active-color: var(--daedalus-primary);
  --cui-sidebar-nav-link-hover-bg: var(--daedalus-primary-trace);
  --cui-sidebar-nav-link-active-bg: rgba(0, 229, 255, 0.10);
  --cui-sidebar-nav-icon-color: var(--daedalus-text-muted);
  --cui-sidebar-nav-icon-hover-color: var(--daedalus-text-secondary);
  --cui-sidebar-nav-icon-active-color: var(--daedalus-primary);
  --cui-sidebar-brand-color: var(--daedalus-text-primary);
  --cui-sidebar-border-color: var(--daedalus-primary-border);

  --cui-header-bg:           var(--daedalus-bg-surface);
  --cui-header-color:        var(--daedalus-text-primary);
  --cui-header-border-color: rgba(0, 229, 255, 0.08);

  --cui-footer-bg:           var(--daedalus-bg-base);
  --cui-footer-color:        var(--daedalus-text-muted);
  --cui-footer-border-color: rgba(0, 229, 255, 0.06);

  --cui-input-bg:            var(--daedalus-bg-surface);
  --cui-input-color:         var(--daedalus-text-primary);
  --cui-input-border-color:  var(--daedalus-text-muted);
  --cui-input-focus-border-color: var(--daedalus-primary);
  --cui-input-placeholder-color: var(--daedalus-text-muted);

  --cui-dropdown-bg:         var(--daedalus-bg-elevated);
  --cui-dropdown-border-color: rgba(0, 229, 255, 0.12);
  --cui-dropdown-link-color: var(--daedalus-text-primary);
  --cui-dropdown-link-hover-bg: var(--daedalus-primary-trace);
  --cui-dropdown-link-hover-color: var(--daedalus-text-primary);

  --cui-table-bg:            transparent;
  --cui-table-striped-bg:    rgba(0, 229, 255, 0.02);
  --cui-table-hover-bg:      rgba(0, 229, 255, 0.06);
  --cui-table-border-color:  rgba(0, 229, 255, 0.08);
  --cui-table-color:         var(--daedalus-text-primary);

  --cui-modal-bg:            var(--daedalus-bg-elevated);
  --cui-modal-header-border-color: rgba(0, 229, 255, 0.10);
  --cui-modal-footer-border-color: rgba(0, 229, 255, 0.10);
}

// ─── Layout structural styles (unchanged from original) ────────────────────

body {
  background-color: var(--daedalus-bg-base);
  font-family: var(--daedalus-font-body);
}

.wrapper {
  width: 100%;
  padding-inline: var(--cui-sidebar-occupy-start, 0) var(--cui-sidebar-occupy-end, 0);
  will-change: auto;
  @include transition(padding .15s);
}

.header > .container-fluid,
.sidebar-header {
  min-height: calc(4rem + 1px); // stylelint-disable-line function-disallowed-list
}

.sidebar-brand-full {
  margin-left: 3px;
}

.sidebar-header {
  .nav-underline-border {
    --cui-nav-underline-border-link-padding-x: 1rem;
    --cui-nav-underline-border-gap: 0;
  }

  .nav-link {
    display: flex;
    align-items: center;
    min-height: calc(4rem + 1px); // stylelint-disable-line function-disallowed-list
  }
}

.sidebar-toggler {
  margin-inline-start: auto;
}

.sidebar-narrow,
.sidebar-narrow-unfoldable:not(:hover) {
  .sidebar-toggler {
    margin-inline-end: auto;
  }
}

.header > .container-fluid + .container-fluid {
  min-height: 3rem;
}

.footer {
  min-height: calc(3rem + 1px); // stylelint-disable-line function-disallowed-list
}

@include color-mode(dark) {
  body {
    background-color: var(--daedalus-bg-base);
  }

  .footer {
    --cui-footer-bg: var(--daedalus-bg-base);
  }
}

// ─── Motion ────────────────────────────────────────────────────────────────

@keyframes fadeSlideUp {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (prefers-reduced-motion: reduce) {
  *,
  *::before,
  *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}
```

### Step 1.3: Verify build passes

Run from workspace root:
```bash
npx nx build daedalus-web-admin
```

Expected: `Successfully ran target build for project daedalus-web-admin` with no SCSS errors.

### Step 1.4: Commit

```bash
git add web-admin/src/index.html web-admin/src/scss/_theme.scss
git commit -m "feat(ui): add Daedalus design tokens and CoreUI overrides"
```

---

## Task 2: Component Overrides (`_custom.scss`)

**Files:**
- Modify: `web-admin/src/scss/_custom.scss`

### Step 2.1: Add component overrides to `_custom.scss`

The current file imports `charts` and `scrollbar` partials — keep those. Append the Daedalus component rules after the existing content.

Replace the entire contents of `web-admin/src/scss/_custom.scss` with:

```scss
// ─── Keep existing partials ─────────────────────────────────────────────────

@use "charts";
@use "scrollbar";

.calendar-cell.today {
  --cui-calendar-cell-today-color: var(--cui-info) !important;
}

.select-week .calendar-row.current {
  cursor: pointer;
}

// ─── Daedalus Component Overrides ───────────────────────────────────────────

// Cards
.card {
  border-radius: 2px;
  border: 1px solid rgba(0, 229, 255, 0.10);
  background-color: var(--daedalus-bg-surface);
  transition: border-color 200ms ease, box-shadow 200ms ease;

  &:hover {
    border-color: rgba(0, 229, 255, 0.30);
    box-shadow: 0 0 20px rgba(0, 229, 255, 0.05);
  }

  .card-header {
    border-radius: 2px 2px 0 0;
    background-color: var(--daedalus-bg-elevated);
    border-bottom: 1px solid rgba(0, 229, 255, 0.08);
    font-family: var(--daedalus-font-display);
    font-size: var(--daedalus-text-sm);
    letter-spacing: 0.05em;
    text-transform: uppercase;
    color: var(--daedalus-text-secondary);
  }

  .card-body {
    font-family: var(--daedalus-font-body);
  }
}

// Tables
.table {
  color: var(--daedalus-text-primary);

  thead th,
  th {
    font-family: var(--daedalus-font-display);
    font-size: var(--daedalus-text-xs);
    letter-spacing: 0.10em;
    text-transform: uppercase;
    color: var(--daedalus-text-secondary);
    border-bottom: 1px solid rgba(0, 229, 255, 0.15) !important;
    padding-top: 0.875rem;
    padding-bottom: 0.875rem;
  }

  tbody tr {
    transition: background-color 120ms ease;

    &:nth-child(even) {
      background-color: rgba(0, 229, 255, 0.02);
    }

    &:hover {
      background-color: rgba(0, 229, 255, 0.06);
      cursor: pointer;
    }
  }

  td {
    border-color: rgba(0, 229, 255, 0.05);
    vertical-align: middle;
    font-family: var(--daedalus-font-body);
    font-size: var(--daedalus-text-base);
  }
}

// Buttons
.btn {
  border-radius: 2px;
  font-family: var(--daedalus-font-body);
  font-size: var(--daedalus-text-sm);
  letter-spacing: 0.02em;
  transition: background-color 120ms ease, border-color 120ms ease, color 120ms ease;
}

.btn-primary {
  background: transparent;
  border: 1px solid var(--daedalus-primary);
  color: var(--daedalus-primary);

  &:hover,
  &:focus {
    background: rgba(0, 229, 255, 0.12);
    border-color: var(--daedalus-primary);
    color: var(--daedalus-primary);
  }

  &:active {
    background: rgba(0, 229, 255, 0.20);
    border-color: var(--daedalus-primary);
    color: var(--daedalus-primary);
  }
}

.btn-danger {
  background: transparent;
  border: 1px solid var(--daedalus-brand);
  color: var(--daedalus-brand);

  &:hover,
  &:focus {
    background: rgba(207, 47, 76, 0.12);
    border-color: var(--daedalus-brand);
    color: var(--daedalus-brand);
  }

  &:active {
    background: rgba(207, 47, 76, 0.20);
    border-color: var(--daedalus-brand);
    color: var(--daedalus-brand);
  }
}

.btn-secondary {
  background: transparent;
  border: 1px solid var(--daedalus-text-muted);
  color: var(--daedalus-text-secondary);

  &:hover,
  &:focus {
    background: var(--daedalus-bg-overlay);
    border-color: var(--daedalus-text-secondary);
    color: var(--daedalus-text-primary);
  }
}

// Form inputs
.form-control,
.form-select {
  border-radius: 2px;
  background-color: var(--daedalus-bg-surface);
  border: 1px solid var(--daedalus-text-muted);
  color: var(--daedalus-text-primary);
  font-family: var(--daedalus-font-body);

  &::placeholder {
    color: var(--daedalus-text-muted);
  }

  &:focus {
    background-color: var(--daedalus-bg-surface);
    border-color: var(--daedalus-primary);
    color: var(--daedalus-text-primary);
    box-shadow: 0 0 0 3px rgba(0, 229, 255, 0.10);
  }
}

.form-label {
  font-family: var(--daedalus-font-display);
  font-size: var(--daedalus-text-xs);
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--daedalus-text-secondary);
  margin-bottom: 0.375rem;
}

// Status badges
.badge-running,
.badge-healthy {
  background: rgba(0, 200, 150, 0.15);
  color: #00C896;
  border: 1px solid rgba(0, 200, 150, 0.30);
  border-radius: 2px;
  font-family: var(--daedalus-font-display);
  font-size: var(--daedalus-text-xs);
  letter-spacing: 0.05em;
  text-transform: uppercase;
  padding: 0.2em 0.6em;
}

.badge-failed,
.badge-critical {
  background: rgba(207, 47, 76, 0.15);
  color: #CF2F4C;
  border: 1px solid rgba(207, 47, 76, 0.30);
  border-radius: 2px;
  font-family: var(--daedalus-font-display);
  font-size: var(--daedalus-text-xs);
  letter-spacing: 0.05em;
  text-transform: uppercase;
  padding: 0.2em 0.6em;
}

.badge-pending,
.badge-degraded {
  background: rgba(255, 179, 0, 0.15);
  color: #FFB300;
  border: 1px solid rgba(255, 179, 0, 0.30);
  border-radius: 2px;
  font-family: var(--daedalus-font-display);
  font-size: var(--daedalus-text-xs);
  letter-spacing: 0.05em;
  text-transform: uppercase;
  padding: 0.2em 0.6em;
}

.badge-info,
.badge-syncing {
  background: rgba(68, 138, 255, 0.15);
  color: #448AFF;
  border: 1px solid rgba(68, 138, 255, 0.30);
  border-radius: 2px;
  font-family: var(--daedalus-font-display);
  font-size: var(--daedalus-text-xs);
  letter-spacing: 0.05em;
  text-transform: uppercase;
  padding: 0.2em 0.6em;
}

// Dashboard stagger animation
.dashboard-widget {
  animation: fadeSlideUp 300ms ease both;

  &:nth-child(1) { animation-delay: 0ms; }
  &:nth-child(2) { animation-delay: 60ms; }
  &:nth-child(3) { animation-delay: 120ms; }
  &:nth-child(4) { animation-delay: 180ms; }
  &:nth-child(5) { animation-delay: 240ms; }
  &:nth-child(6) { animation-delay: 300ms; }
}

// Modals
.modal-content {
  border-radius: 2px;
  background-color: var(--daedalus-bg-elevated);
  border: 1px solid rgba(0, 229, 255, 0.12);
}

.modal-header {
  border-bottom: 1px solid rgba(0, 229, 255, 0.10);
  font-family: var(--daedalus-font-display);
  font-size: var(--daedalus-text-sm);
  letter-spacing: 0.05em;
  text-transform: uppercase;
}

.modal-footer {
  border-top: 1px solid rgba(0, 229, 255, 0.10);
}

// Dropdowns
.dropdown-menu {
  border-radius: 2px;
  background-color: var(--daedalus-bg-elevated);
  border: 1px solid rgba(0, 229, 255, 0.12);
}

.dropdown-item {
  font-family: var(--daedalus-font-body);
  color: var(--daedalus-text-primary);
  font-size: var(--daedalus-text-sm);

  &:hover,
  &:focus {
    background-color: var(--daedalus-primary-trace);
    color: var(--daedalus-text-primary);
  }
}

// Breadcrumbs
.breadcrumb {
  font-family: var(--daedalus-font-body);
  font-size: var(--daedalus-text-sm);

  .breadcrumb-item {
    color: var(--daedalus-text-muted);

    &.active {
      color: var(--daedalus-text-primary);
    }

    + .breadcrumb-item::before {
      color: var(--daedalus-primary-dim);
    }
  }

  a {
    color: var(--daedalus-text-secondary);
    text-decoration: none;

    &:hover {
      color: var(--daedalus-primary);
    }
  }
}

// Pagination
.page-link {
  background-color: var(--daedalus-bg-surface);
  border: 1px solid rgba(0, 229, 255, 0.10);
  color: var(--daedalus-text-secondary);
  border-radius: 2px;
  font-family: var(--daedalus-font-display);
  font-size: var(--daedalus-text-xs);

  &:hover {
    background-color: var(--daedalus-primary-trace);
    color: var(--daedalus-primary);
    border-color: rgba(0, 229, 255, 0.20);
  }
}

.page-item.active .page-link {
  background-color: transparent;
  border-color: var(--daedalus-primary);
  color: var(--daedalus-primary);
}
```

### Step 2.2: Verify build passes

```bash
npx nx build daedalus-web-admin
```

Expected: build succeeds with no SCSS errors.

### Step 2.3: Commit

```bash
git add web-admin/src/scss/_custom.scss
git commit -m "feat(ui): add Daedalus component overrides — cards, tables, buttons, badges"
```

---

## Task 3: Sidebar Shell

**Files:**
- Modify: `web-admin/src/app/layout/default-layout/default-layout.component.html`
- Modify: `web-admin/src/app/layout/default-layout/default-layout.component.scss`

### Step 3.1: Add scanline wrapper to sidebar HTML

The scanline is a `::after` pseudo-element on the sidebar. CoreUI renders the sidebar as a `<c-sidebar>` component — we add a wrapping `div.daedalus-sidebar-wrap` to host the pseudo-element.

Replace the entire contents of `web-admin/src/app/layout/default-layout/default-layout.component.html` with:

```html
<!--sidebar-->
<div class="daedalus-sidebar-wrap">
  <c-sidebar
    #sidebar1="cSidebar"
    class="d-print-none sidebar sidebar-fixed border-end"
    colorScheme="dark"
    id="sidebar1"
    visible
  >
    <c-sidebar-header class="border-bottom">
      <c-sidebar-brand [routerLink]="[]">
        <img class="sidebar-brand-full" src="assets/brand/Daedalous.svg" height="32" alt="Daedalus Logo" />
        <img class="sidebar-brand-narrow" src="assets/brand/Daedalous.svg" height="32" alt="Daedalus Logo" />
      </c-sidebar-brand>
    </c-sidebar-header>

    <ng-scrollbar pointerEventsMethod="scrollbar" visibility="hover">
      <c-sidebar-nav [navItems]="navItems" dropdownMode="close" compact />
    </ng-scrollbar>

    @if (!sidebar1.narrow) {
      <c-sidebar-footer cSidebarToggle="sidebar1" class="border-top d-none d-lg-flex" toggle="unfoldable" style="cursor: pointer;">
        <button cSidebarToggler aria-label="Toggle sidebar fold"></button>
      </c-sidebar-footer>
    }
  </c-sidebar>
</div>

<!--main-->
<div class="wrapper d-flex flex-column min-vh-100">
  <!--app-header-->
  <app-default-header
    [cShadowOnScroll]="'sm'"
    class="mb-4 d-print-none header header-sticky p-0 shadow-sm"
    position="sticky"
    sidebarId="sidebar1"
  />
  <!--app-body-->
  <div class="body flex-grow-1">
    <c-container breakpoint="lg" class="h-auto px-4">
      <router-outlet />
    </c-container>
  </div>
  <!--app footer-->
  <app-default-footer />
</div>
```

### Step 3.2: Rewrite sidebar SCSS

Replace the entire contents of `web-admin/src/app/layout/default-layout/default-layout.component.scss` with:

```scss
// ─── Scanline wrapper ───────────────────────────────────────────────────────
// The ::after pseudo-element creates a CRT-like texture over the sidebar.
// pointer-events: none ensures it doesn't block sidebar interactions.

.daedalus-sidebar-wrap {
  position: relative;

  &::after {
    content: '';
    position: fixed;
    top: 0;
    left: 0;
    width: var(--cui-sidebar-width, 256px);
    height: 100vh;
    background: repeating-linear-gradient(
      0deg,
      transparent,
      transparent 1px,
      rgba(0, 0, 0, 0.12) 1px,
      rgba(0, 0, 0, 0.12) 2px
    );
    pointer-events: none;
    z-index: 1100;
    transition: width 0.15s;
  }
}

// ─── Sidebar nav active indicator ──────────────────────────────────────────

:host::ng-deep {
  .sidebar-nav .nav-link {
    position: relative;
    font-family: var(--daedalus-font-body);
    font-size: 0.875rem;
    transition: color 120ms ease, background-color 120ms ease;

    &::before {
      content: '';
      position: absolute;
      left: 0;
      top: 0;
      height: 100%;
      width: 0;
      background-color: var(--daedalus-primary);
      transition: width 80ms ease;
      border-radius: 0 1px 1px 0;
    }

    &.active::before {
      width: 3px;
    }
  }

  // Scrollbar theming
  .ng-scrollbar {
    --scrollbar-padding: 1px;
    --scrollbar-size: 4px;
    --scrollbar-thumb-color: rgba(0, 229, 255, 0.20);
    --scrollbar-thumb-hover-color: rgba(0, 229, 255, 0.40);
    --scrollbar-hover-size: calc(var(--scrollbar-size) * 1.5);
    --scrollbar-border-radius: 2px;
  }

  .ng-scroll-content {
    display: flex;
    min-height: 100%;
  }
}
```

### Step 3.3: Verify build passes

```bash
npx nx build daedalus-web-admin
```

Expected: build succeeds. If `::ng-deep` triggers a lint warning, it is acceptable here — it is the only reliable way to style CoreUI sidebar internals from the host component.

### Step 3.4: Commit

```bash
git add web-admin/src/app/layout/default-layout/default-layout.component.html \
        web-admin/src/app/layout/default-layout/default-layout.component.scss
git commit -m "feat(ui): redesign sidebar — scanline texture, active indicator, cian scrollbar"
```

---

## Task 4: Header Cleanup + Styling

**Files:**
- Modify: `web-admin/src/app/layout/default-layout/default-header/default-header.component.ts`
- Modify: `web-admin/src/app/layout/default-layout/default-header/default-header.component.html` (minor)
- Modify: `web-admin/src/app/layout/default-layout/default-header/default-header.component.scss`

### Step 4.1: Remove unused demo data from header component TS

The current `default-header.component.ts` contains `newMessages`, `newNotifications`, `newStatus`, and `newTasks` arrays that are not used in the template. Remove them. Also remove unused imports: `BadgeComponent`, `DropdownDividerDirective`, `DropdownHeaderDirective`, `NavItemComponent`, `NavLinkDirective`, `RouterLink`, `RouterLinkActive`.

Replace the entire contents of `web-admin/src/app/layout/default-layout/default-header/default-header.component.ts` with:

```typescript
import { NgTemplateOutlet } from '@angular/common';
import { Component, inject, input } from '@angular/core';
import { AuthService } from '../../../auth/auth.service';

import {
  AvatarComponent,
  BreadcrumbRouterComponent,
  ContainerComponent,
  DropdownComponent,
  DropdownItemDirective,
  DropdownMenuDirective,
  DropdownToggleDirective,
  HeaderComponent,
  HeaderNavComponent,
  HeaderTogglerDirective,
  SidebarToggleDirective
} from '@coreui/angular';

import { IconDirective } from '@coreui/icons-angular';

@Component({
  selector: 'app-default-header',
  templateUrl: './default-header.component.html',
  imports: [
    ContainerComponent,
    HeaderTogglerDirective,
    SidebarToggleDirective,
    IconDirective,
    HeaderNavComponent,
    NgTemplateOutlet,
    BreadcrumbRouterComponent,
    DropdownComponent,
    DropdownToggleDirective,
    AvatarComponent,
    DropdownMenuDirective,
    DropdownItemDirective
  ]
})
export class DefaultHeaderComponent extends HeaderComponent {
  readonly #authService = inject(AuthService);

  constructor() {
    super();
  }

  sidebarId = input('sidebar1');

  logout() {
    this.#authService.logout();
  }
}
```

### Step 4.2: Add CSS class to header toggler in HTML

The header HTML is already clean. Add the `daedalus-header-toggler` class to the toggle button for styling:

Replace the entire contents of `web-admin/src/app/layout/default-layout/default-header/default-header.component.html` with:

```html
<ng-container>
  <c-container [fluid]="true" class="border-bottom px-4">
    <button
      [cSidebarToggle]="sidebarId()"
      cHeaderToggler
      class="btn daedalus-header-toggler"
      toggle="visible"
      style="margin-inline-start: -14px;"
      aria-label="Toggle sidebar navigation"
    >
      <svg cIcon name="cilMenu" size="lg"></svg>
    </button>

    <c-header-nav class="mx-0">
      <ng-container *ngTemplateOutlet="userDropdown" />
    </c-header-nav>
  </c-container>

  <c-container [fluid]="true" class="px-4">
    <c-breadcrumb-router />
  </c-container>
</ng-container>

<ng-template #userDropdown>
  <c-dropdown [popperOptions]="{ placement: 'bottom-start' }" variant="nav-item">
    <button [caret]="false" cDropdownToggle class="py-0 pe-0 daedalus-avatar-btn" aria-label="Open user menu">
      <c-avatar
        shape="rounded-1"
        size="md"
        src="./assets/images/avatars/Icon-avatar.png"
        status="success"
        textColor="primary"
        alt="User avatar"
      />
    </button>
    <ul cDropdownMenu class="pt-0 w-auto">
      <li>
        <button cDropdownItem (click)="logout()">
          <svg cIcon class="me-2" name="cilAccountLogout"></svg>
          Logout
        </button>
      </li>
    </ul>
  </c-dropdown>
</ng-template>
```

### Step 4.3: Write header SCSS

Replace the entire contents of `web-admin/src/app/layout/default-layout/default-header/default-header.component.scss` with:

```scss
:host {
  background-color: var(--daedalus-bg-surface);
  border-bottom: 1px solid rgba(0, 229, 255, 0.08);
}

:host::ng-deep {
  .daedalus-header-toggler {
    color: var(--daedalus-text-secondary);
    transition: color 120ms ease;

    &:hover {
      color: var(--daedalus-primary);
      background: transparent;
    }
  }

  .daedalus-avatar-btn {
    c-avatar {
      outline: 2px solid rgba(0, 229, 255, 0.0);
      border-radius: 4px;
      transition: outline-color 150ms ease;
    }

    &:hover c-avatar {
      outline-color: rgba(0, 229, 255, 0.40);
    }
  }

  .breadcrumb {
    padding: 0.5rem 0;
    margin-bottom: 0;
  }
}
```

### Step 4.4: Verify build passes

```bash
npx nx build daedalus-web-admin
```

Expected: build succeeds. There should be no TypeScript errors from the removed unused properties.

### Step 4.5: Commit

```bash
git add web-admin/src/app/layout/default-layout/default-header/default-header.component.ts \
        web-admin/src/app/layout/default-layout/default-header/default-header.component.html \
        web-admin/src/app/layout/default-layout/default-header/default-header.component.scss
git commit -m "feat(ui): redesign header — remove demo data, add Daedalus styling"
```

---

## Task 5: Footer Redesign

**Files:**
- Modify: `web-admin/src/app/layout/default-layout/default-footer/default-footer.component.ts`
- Modify: `web-admin/src/app/layout/default-layout/default-footer/default-footer.component.html`
- Modify: `web-admin/src/app/layout/default-layout/default-footer/default-footer.component.scss`

### Step 5.1: Add version constant to footer component TS

Replace the entire contents of `web-admin/src/app/layout/default-layout/default-footer/default-footer.component.ts` with:

```typescript
import { Component } from '@angular/core';
import { FooterComponent } from '@coreui/angular';

@Component({
  selector: 'app-default-footer',
  templateUrl: './default-footer.component.html',
  styleUrls: ['./default-footer.component.scss']
})
export class DefaultFooterComponent extends FooterComponent {
  readonly version = 'v0.1.0';

  constructor() {
    super();
  }
}
```

### Step 5.2: Rewrite footer HTML

Replace the entire contents of `web-admin/src/app/layout/default-layout/default-footer/default-footer.component.html` with:

```html
<div class="daedalus-footer">
  <span class="daedalus-footer__brand">Daedalus Orchestrator</span>
  <span class="daedalus-footer__copy">&copy; 2026</span>
  <span class="daedalus-footer__version">{{ version }}</span>
</div>
```

### Step 5.3: Write footer SCSS

Replace the entire contents of `web-admin/src/app/layout/default-layout/default-footer/default-footer.component.scss` with:

```scss
:host {
  display: block;
  background-color: var(--daedalus-bg-base);
  border-top: 1px solid rgba(0, 229, 255, 0.06);
}

.daedalus-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  position: relative;
  padding: 0.75rem 1.5rem;
  min-height: calc(3rem + 1px);
}

.daedalus-footer__brand {
  font-family: var(--daedalus-font-body);
  font-size: var(--daedalus-text-sm);
  color: var(--daedalus-text-muted);
}

.daedalus-footer__copy {
  font-family: var(--daedalus-font-body);
  font-size: var(--daedalus-text-sm);
  color: var(--daedalus-text-muted);
  position: absolute;
  left: 50%;
  transform: translateX(-50%);
}

.daedalus-footer__version {
  font-family: var(--daedalus-font-display);
  font-size: var(--daedalus-text-xs);
  color: var(--daedalus-primary);
  letter-spacing: 0.08em;
}
```

### Step 5.4: Verify build passes

```bash
npx nx build daedalus-web-admin
```

Expected: build succeeds. `{{ version }}` should be bound correctly since the property is `readonly version` on the component class.

### Step 5.5: Commit

```bash
git add web-admin/src/app/layout/default-layout/default-footer/default-footer.component.ts \
        web-admin/src/app/layout/default-layout/default-footer/default-footer.component.html \
        web-admin/src/app/layout/default-layout/default-footer/default-footer.component.scss
git commit -m "feat(ui): redesign footer — brand + version + copyright in Daedalus style"
```

---

## Task 6: Final Visual Verification

### Step 6.1: Serve the app

```bash
npx nx serve daedalus-web-admin
```

Open `http://localhost:4200` in the browser.

### Step 6.2: Visual checklist

Check each of the following:

- [ ] **Background** is deep near-black `#080C0F` (not white, not blue)
- [ ] **Sidebar** has a subtle cian right-border glow and scanline texture
- [ ] **Active nav item** shows a 3px cian left border + cian text
- [ ] **Inactive nav items** are muted gray, hover brightens them
- [ ] **Header** has dark surface background, subtle bottom border
- [ ] **Header toggler** turns cian on hover
- [ ] **Breadcrumb** separators are cian-dim colored
- [ ] **Footer** shows brand name (left), `©` (center), version in cian (right)
- [ ] **Cards** have near-invisible cian border that intensifies on hover
- [ ] **Fonts** are Geist Mono (display) and IBM Plex Sans (body) — not system fonts
- [ ] **Tables** have monospace uppercase headers with letter-spacing

### Step 6.3: Final commit

```bash
git add -A
git commit -m "feat(ui): complete Daedalus industrial UI redesign — dark + cian + crimson identity"
```

---

## Spec Coverage Check

| Spec section | Covered by task |
|---|---|
| Color tokens (Daedalus + CoreUI overrides) | Task 1 |
| Google Fonts (Geist Mono + IBM Plex Sans) | Task 1 |
| Cards, tables, buttons, inputs, badges | Task 2 |
| Motion (fadeSlideUp keyframe, prefers-reduced-motion) | Task 1 |
| Sidebar scanline, active indicator, scrollbar | Task 3 |
| Header cleanup, toggler style, avatar border | Task 4 |
| Footer brand + version + copyright | Task 5 |
| Dashboard stagger animation class | Task 2 |
| Modals, dropdowns, breadcrumbs, pagination | Task 2 |
