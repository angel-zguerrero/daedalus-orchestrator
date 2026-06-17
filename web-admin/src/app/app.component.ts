import { Component, DestroyRef, inject, OnInit } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { Title } from '@angular/platform-browser';
import { ActivatedRoute, NavigationEnd, Router, RouterOutlet } from '@angular/router';
import { delay, filter, map, tap } from 'rxjs/operators';

import { ColorModeService, AlertComponent } from '@coreui/angular';
import { IconSetService } from '@coreui/icons-angular';
import { iconSubset } from './icons/icon-subset';
import { AuthService } from './auth/auth.service';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-root',
  template: `
    @if (authService.isDemo$ | async) {
      <div style="background-color: #ffc107; color: #000; text-align: center; padding: 10px; font-weight: bold; z-index: 1050; position: relative;">
        Modo demo, para acceder usuar usuario admin, password admin
      </div>
    }
    <router-outlet />
  `,
  imports: [RouterOutlet, CommonModule]
})
export class AppComponent implements OnInit {
  title = 'Daedalus Orchestrator';
  authService = inject(AuthService);

  readonly #destroyRef: DestroyRef = inject(DestroyRef);
  readonly #activatedRoute: ActivatedRoute = inject(ActivatedRoute);
  readonly #router = inject(Router);
  readonly #titleService = inject(Title);

  readonly #colorModeService = inject(ColorModeService);
  readonly #iconSetService = inject(IconSetService);

  constructor() {
    this.#titleService.setTitle(this.title);
    // iconSet singleton
    this.#iconSetService.icons = { ...iconSubset };
    this.#colorModeService.localStorageItemName.set('daedalus-web-admin-theme-default');
    this.#colorModeService.eventName.set('ColorSchemeChange');
    this.#colorModeService.colorMode.set('dark');
  }

  ngOnInit(): void {
    this.#router.events.pipe(
      filter(event => event instanceof NavigationEnd),
      map(() => {
        let route = this.#activatedRoute;
        while (route.firstChild) {
          route = route.firstChild;
        }
        return route;
      }),
      filter(route => route.outlet === 'primary'),
      map(route => route.snapshot.data['title']),
      filter(title => !!title),
      takeUntilDestroyed(this.#destroyRef)
    ).subscribe((title: string) => {
      this.#titleService.setTitle(`Daedalus - ${title}`);
    });
  }
}
