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
    name: 'Cluster Management',
    url: '/cluster',
    iconComponent: { name: 'cil-layers' }
  },
  {
    name: 'Node Schedulers',
    url: '/node-schedulers',
    iconComponent: { name: 'cil-task' }
  }
];
