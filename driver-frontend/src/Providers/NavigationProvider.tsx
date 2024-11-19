import { JoinPaths } from '@src/Utils/path_utils';
import { useNavigate } from 'react-router-dom';

function isNumber(value?: string | string[] | number): boolean {
    return value != null && value !== '' && !Array.isArray(value) && !isNaN(Number(value.toString()));
}

function useNavigation() {
    const navigate = useNavigate();

    const doNavigate = (paths: number | string | string[] = '') => {
        if (isNumber(paths)) {
            navigate(paths as number);
            return;
        }

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
