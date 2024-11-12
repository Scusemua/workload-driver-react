import { AuthorizationContext } from '@Providers/AuthProvider';
import { JoinPaths } from '@src/Utils/path_utils';
import React from 'react';
import { Navigate, useLocation } from 'react-router-dom';

function PrivateRoute({ children, nextPath }: { children: React.JSX.Element; nextPath: string }) {
    const { authenticated } = React.useContext(AuthorizationContext);

    const location = useLocation();

    if (!authenticated) {
        const loginPath: string = JoinPaths(process.env.PUBLIC_PATH || '/', 'login');
        return <Navigate to={loginPath} state={{ from: location, nextPath: nextPath }} />;
    }

    return children;
}

export default PrivateRoute;
