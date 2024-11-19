import { DashboardLoginPage } from '@App/DashboardLoginPage';
import { IndividualKernelsPage } from '@App/KernelsPage';
import { KernelsPage } from '@App/KernelsPage/KernelsPage';
import { NodesPage } from '@App/NodesPage/NodesPage';
import PrivateRoute from '@App/PrivateRoute/PrivateRoute';
import IndividualWorkloadPage from '@App/WorkloadsPage/IndividualWorkloadPage';
import { RegisterWorkloadPage } from '@App/WorkloadsPage/RegisterWorkloadPage';
import { Dashboard } from '@src/app/Dashboard/Dashboard';
import { NotFound } from '@src/app/NotFound/NotFound';
import { WorkloadsPage } from '@src/app/WorkloadsPage/WorkloadsPage';
import { JoinPaths } from '@src/Utils/path_utils';
import * as React from 'react';
import { Route, Routes } from 'react-router-dom';

// a custom hook for setting the page title
export function useDocumentTitle(title: string) {
    React.useEffect(() => {
        const originalTitle = document.title;
        document.title = title;

        return () => {
            document.title = originalTitle;
        };
    }, [title]);
}

export interface IAppRoute {
    label?: string; // Excluding the label will exclude the route from the nav sidebar in AppLayout
    /* eslint-disable @typescript-eslint/no-explicit-any */
    component: React.ComponentType<any>;
    /* eslint-enable @typescript-eslint/no-explicit-any */
    exact?: boolean;
    path: string;
    title: string;
    routes?: undefined;
    isPrivate?: boolean;
}

export interface IAppRouteGroup {
    label: string;
    routes: IAppRoute[];
}

export type AppRouteConfig = IAppRoute | IAppRouteGroup;

const routes: AppRouteConfig[] = [
    {
        component: DashboardLoginPage,
        exact: true,
        path: JoinPaths(process.env.PUBLIC_PATH || '/', 'login'),
        title: 'Login page',
        isPrivate: false,
    },
    {
        component: Dashboard,
        exact: true,
        label: 'Main Dashboard',
        path: JoinPaths(process.env.PUBLIC_PATH || '/'),
        title: 'Main Dashboard',
        isPrivate: true,
    },
    {
        label: 'Specific Pages',
        routes: [
            {
                component: WorkloadsPage,
                exact: true,
                label: 'Workloads',
                path: JoinPaths(process.env.PUBLIC_PATH || '/', 'workloads'),
                title: 'Workloads',
                isPrivate: true,
            },
            {
                component: KernelsPage,
                exact: true,
                label: 'Kernels',
                path: JoinPaths(process.env.PUBLIC_PATH || '/', '/kernels'),
                title: 'Kernels',
                isPrivate: true,
            },
            {
                component: NodesPage,
                exact: true,
                label: 'Nodes',
                path: JoinPaths(process.env.PUBLIC_PATH || '/', '/nodes'),
                title: 'Nodes',
                isPrivate: true,
            },
        ],
    },
    {
        component: IndividualWorkloadPage,
        exact: true,
        path: JoinPaths(process.env.PUBLIC_PATH || '/', '/workload/:workload_id'),
        title: 'Workload',
        isPrivate: true,
    },
    {
        component: IndividualKernelsPage,
        exact: true,
        path: JoinPaths(process.env.PUBLIC_PATH || '/', '/kernels/:kernel_id'),
        title: 'Kernel',
        isPrivate: true,
    },
    {
        component: RegisterWorkloadPage,
        exact: true,
        path: JoinPaths(process.env.PUBLIC_PATH || '/', '/register_workload'),
        title: 'Register a New Workload',
        isPrivate: true,
    },
];

const getRoute = (props: IAppRoute) => {
    if (props.isPrivate) {
        return (
            <Route
                key={`private-route-${props.path}`}
                path={props.path}
                element={
                    <PrivateRoute nextPath={props.path}>
                        <props.component />
                    </PrivateRoute>
                }
            />
        );
    } else {
        return <Route key={`route-${props.path}`} path={props.path} element={<props.component />} />;
    }
};

const flattenedRoutes: IAppRoute[] = routes.reduce(
    (flattened, route) => [...flattened, ...(route.routes ? route.routes : [route])],
    [] as IAppRoute[],
);

const AppRoutes = (): React.ReactElement => (
    <Routes>
        {flattenedRoutes.map((props: IAppRoute) => getRoute(props))}
        <Route key={'route-not-found'} element={<NotFound />} />
    </Routes>
);

export { AppRoutes, routes };
