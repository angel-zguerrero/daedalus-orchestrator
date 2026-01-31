import { Component, DestroyRef, inject, OnInit } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { Title } from '@angular/platform-browser';
import { ActivatedRoute, NavigationEnd, Router, RouterOutlet } from '@angular/router';
import { delay, filter, map, tap } from 'rxjs/operators';

import { ColorModeService } from '@coreui/angular';
import { IconSetService } from '@coreui/icons-angular';
import { iconSubset } from './icons/icon-subset';

@Component({
  selector: 'app-root',
  template: '<router-outlet />',
  imports: [RouterOutlet]
})
export class AppComponent implements OnInit {
  title = 'Daedalus Orchestrator';

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
