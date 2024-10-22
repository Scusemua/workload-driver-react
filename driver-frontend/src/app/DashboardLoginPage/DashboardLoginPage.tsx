import { ListItem, LoginFooterItem, LoginForm } from '@patternfly/react-core';
import { BackgroundImage } from '@patternfly/react-core/src/components/BackgroundImage';
import { Brand } from '@patternfly/react-core/src/components/Brand';
import { List, ListVariant } from '@patternfly/react-core/src/components/List';
import { Login } from '@patternfly/react-core/src/components/LoginPage/Login';
import { LoginFooter } from '@patternfly/react-core/src/components/LoginPage/LoginFooter';
import { LoginHeader } from '@patternfly/react-core/src/components/LoginPage/LoginHeader';
import { LoginMainBody } from '@patternfly/react-core/src/components/LoginPage/LoginMainBody';
import { LoginMainFooter } from '@patternfly/react-core/src/components/LoginPage/LoginMainFooter';
import { LoginMainHeader } from '@patternfly/react-core/src/components/LoginPage/LoginMainHeader';
import { ExternalLinkAltIcon, GithubIcon } from '@patternfly/react-icons';
import ExclamationCircleIcon from '@patternfly/react-icons/dist/esm/icons/exclamation-circle-icon';
import { css } from '@patternfly/react-styles';
import { AuthorizationContext } from '@Providers/AuthProvider';
import logo_greyscale from '@src/app/bgimages/icon_greyscale.svg';
import logo from '@src/app/bgimages/WorkloadDriver-Logo.svg';
import React from 'react';

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
            await mutateToken(username, password).catch((err: Error) => {
                console.error(`Failed to login: ${err}`);

                setIsValidPassword(false);
                setIsValidPassword(false);
                setShowHelperText(true);
            });
        }
    };

    // icon={<Ds2Icon scale={1.5} />}
    const footerListItems = (
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

    const HeaderBrand = (
        <React.Fragment>
            <Brand src={logo} alt={'Distributed Dashboard Logo'} />
        </React.Fragment>
    );
    const Header = <LoginHeader headerBrand={HeaderBrand} />;
    const Footer = (
        <LoginFooter>
            <p>{'Distributed Notebook Cluster | Admin Dashboard & Workload Orchestrator'}</p>
            <List variant={ListVariant.inline}>{footerListItems}</List>
        </LoginFooter>
    );

    return (
        <React.Fragment>
            {logo_greyscale && <BackgroundImage src={logo_greyscale} />}
            <Login header={Header} footer={Footer} className={css('gradient_background')} {...props}>
                <LoginMainHeader
                    title={'Log in to access the Dashboard'}
                    subtitle={'Enter the configured admin credentials'}
                />
                <LoginMainBody>{loginForm}</LoginMainBody>
                <LoginMainFooter />
            </Login>
        </React.Fragment>
        // <LoginPage
        //     footerListVariants={ListVariant.inline}
        //     brandImgSrc={logo}
        //     brandImgAlt="Distributed Dashboard Logo"
        //     backgroundImgSrc={logo_greyscale}
        //     footerListItems={listItem}
        //     textContent="Distributed Notebook Cluster | Admin Dashboard & Workload Orchestrator"
        //     loginTitle="Log in to access the Dashboard"
        //     loginSubtitle="Enter the configured admin credentials"
        // >
        //     {loginForm}
        // </LoginPage>
    );
};
