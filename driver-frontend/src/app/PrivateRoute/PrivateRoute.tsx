import { AuthorizationContext } from '@Providers/AuthProvider';
import React from 'react';
import { Navigate, useLocation } from 'react-router-dom';

function PrivateRoute({ children, nextPath }: { children: React.JSX.Element, nextPath: string }) {
    const { authenticated } = React.useContext(AuthorizationContext);

    const location = useLocation();

    if (!authenticated) {
        return <Navigate to="/login" state={{ from: location, nextPath: nextPath }} />;
    }

    return children;
}

export default PrivateRoute;
