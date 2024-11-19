import { JoinPaths } from '@src/Utils/path_utils';
import { useNavigate } from 'react-router-dom';

function useNavigation() {
    const navigate = useNavigate();

    const doNavigate = (paths: string | string[] = '') => {
        let path: string = process.env.PUBLIC_PATH || '/';
        if (Array.isArray(paths)) {
            path = JoinPaths(path, ...paths);
        } else {
            path = JoinPaths(path, paths as string);
        }

        navigate(path);
    };

    return {
        navigate: doNavigate,
    };
}

export default useNavigation;
