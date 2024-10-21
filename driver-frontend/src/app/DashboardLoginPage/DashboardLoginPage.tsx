import { ListItem, ListVariant, LoginFooterItem, LoginForm, LoginPage } from '@patternfly/react-core';
import { ExternalLinkAltIcon, GithubIcon } from '@patternfly/react-icons';
import ExclamationCircleIcon from '@patternfly/react-icons/dist/esm/icons/exclamation-circle-icon';
import { AuthorizationContext } from '@Providers/AuthProvider';
import logo_greyscale from '@src/app/bgimages/icon_greyscale.svg';
import logo from '@src/app/bgimages/WorkloadDriver-Logo.svg';
import React from 'react';
import { toast } from 'react-hot-toast';

interface DashboardLoginPageProps {
    onSuccessfulLogin: (token: string, expiration: string) => void;
}

export const DashboardLoginPage: React.FunctionComponent<DashboardLoginPageProps> = (
    props: DashboardLoginPageProps,
) => {
    const [showHelperText, setShowHelperText] = React.useState<boolean>(false);
    const [username, setUsername] = React.useState<string>('');
    const [isValidUsername, setIsValidUsername] = React.useState<boolean>(true);
    const [password, setPassword] = React.useState<string>('');
    const [isValidPassword, setIsValidPassword] = React.useState<boolean>(true);

    // username, setUsername, password, setPassword,
    const { mutateToken, error } = React.useContext(AuthorizationContext);

    React.useEffect(() => {
        if (error) {
            setIsValidPassword(false);
            setIsValidUsername(false);
            setShowHelperText(true);
        } else {
            setIsValidPassword(true);
            setIsValidUsername(true);
            setShowHelperText(false);
        }
    }, [error]);

    const handleUsernameChange = (_event: React.FormEvent<HTMLInputElement>, value: string) => {
        setUsername(value);
    };

    const handlePasswordChange = (_event: React.FormEvent<HTMLInputElement>, value: string) => {
        setPassword(value);
    };

    const onLoginButtonClick = async (event: React.MouseEvent<HTMLButtonElement, MouseEvent>) => {
        event.preventDefault();

        if (mutateToken) {
            const toastId: string = toast.loading('Attempting to log in...');
            await mutateToken(username, password)
                .catch((err: Error) => {
                    console.error(`Failed to login: ${err}`);

                    setIsValidPassword(false);
                    setIsValidPassword(false);
                    setShowHelperText(true);

                    toast.dismiss(toastId);
                })
                .then(() => {
                    toast.success('Authenticated. You have been logged in.', { id: toastId });
                });
        }
    };

    // icon={<Ds2Icon scale={1.5} />}
    const listItem = (
        <React.Fragment>
            <ListItem icon={<GithubIcon />}>
                <LoginFooterItem href="https://github.com/Scusemua/workload-driver-react">Source Code</LoginFooterItem>
            </ListItem>
            <ListItem icon={<ExternalLinkAltIcon />}>
                <LoginFooterItem href="https://ds2-lab.github.io/">
                    D<span className="lowerc">S</span>
                    <sup>2</sup> Research Lab @ UVA
                </LoginFooterItem>
            </ListItem>
        </React.Fragment>
    );

    const loginForm = (
        <LoginForm
            showHelperText={showHelperText}
            helperText="Invalid login credentials."
            helperTextIcon={<ExclamationCircleIcon />}
            usernameLabel="Username"
            usernameValue={username}
            onChangeUsername={handleUsernameChange}
            isValidUsername={isValidUsername}
            passwordLabel="Password"
            passwordValue={password}
            onChangePassword={handlePasswordChange}
            isShowPasswordEnabled
            isValidPassword={isValidPassword}
            isLoginButtonDisabled={username === '' || password === ''}
            onLoginButtonClick={onLoginButtonClick}
            loginButtonLabel="Log in"
        />
    );

    return (
        <LoginPage
            footerListVariants={ListVariant.inline}
            brandImgSrc={logo}
            brandImgAlt="Distributed Dashboard Logo"
            backgroundImgSrc={logo_greyscale}
            footerListItems={listItem}
            textContent="Distributed Notebook Cluster | Admin Dashboard & Workload Orchestrator"
            loginTitle="Log in to access the Dashboard"
            loginSubtitle="Enter the configured admin credentials"
        >
            {loginForm}
        </LoginPage>
    );
};
