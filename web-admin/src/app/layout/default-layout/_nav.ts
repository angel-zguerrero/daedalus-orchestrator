import { INavData } from '@coreui/angular';

export const navItems: INavData[] = [
  {
    name: 'Dashboard',
    url: '/dashboard',
    iconComponent: { name: 'cil-speedometer' }
  },
  {
    name: 'Tenants',
    url: '/tenants',
    iconComponent: { name: 'cil-people' }
  },
  {
    name: 'Node Schedulers',
    url: '/node-schedulers',
    iconComponent: { name: 'cil-task' }
  }
];
