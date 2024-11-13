import { ListItem, LoginFooterItem, LoginForm, LoginPage } from '@patternfly/react-core';
import { ExternalLinkAltIcon } from '@patternfly/react-icons';
import GithubIcon from '@patternfly/react-icons/dist/dynamic/icons/github-icon';
import ExclamationCircleIcon from '@patternfly/react-icons/dist/esm/icons/exclamation-circle-icon';
import { AuthorizationContext } from '@Providers/AuthProvider';
import logo_greyscale from '@src/app/bgimages/icon_greyscale.svg';
import logo from '@src/app/bgimages/WorkloadDriver-Logo.svg';
import { JoinPaths } from '@src/Utils/path_utils';
import React from 'react';
import { useLocation, useNavigate } from 'react-router-dom';

export const DashboardLoginPage: React.FunctionComponent = () => {
  const [showHelperText, setShowHelperText] = React.useState<boolean>(false);
  const [username, setUsername] = React.useState<string>('');
  const [isValidUsername, setIsValidUsername] = React.useState<boolean>(true);
  const [password, setPassword] = React.useState<string>('');
  const [isValidPassword, setIsValidPassword] = React.useState<boolean>(true);

  // username, setUsername, password, setPassword,
  const { mutateToken, error } = React.useContext(AuthorizationContext);

  const navigate = useNavigate();
  const location = useLocation();

  const { authenticated } = React.useContext(AuthorizationContext);

  React.useEffect(() => {
    if (authenticated) {
      let nextPath: string | null = null;

      if (location !== null && location.state !== null) {
        nextPath = location.state.nextPath;
      }

      if (nextPath === null || nextPath === '') {
        nextPath = JoinPaths(process.env.PUBLIC_PATH || '/');
      }

      navigate(nextPath);
    }
  }, [authenticated, location, location.state, navigate]);

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

    if (username === '') {
      setIsValidUsername(false);
      setShowHelperText(true);
      return;
    } else if (password === '') {
      setIsValidPassword(false);
      setShowHelperText(true);
      return;
    }

    if (mutateToken) {
      await mutateToken(username, password).catch((err: Error) => {
        console.error(`Failed to login: ${err}`);

        setIsValidUsername(false);
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

  // const loginForm = (
  //   <LoginForm
  //     showHelperText={showHelperText}
  //     helperText="Invalid login credentials."
  //     helperTextIcon={<ExclamationCircleIcon />}
  //     usernameLabel="Username"
  //     usernameValue={username}
  //     onChangeUsername={handleUsernameChange}
  //     isValidUsername={isValidUsername}
  //     passwordLabel="Password"
  //     passwordValue={password}
  //     onChangePassword={handlePasswordChange}
  //     isValidPassword={isValidPassword}
  //     rememberMeLabel="Keep me logged in for 30 days."
  //     onLoginButtonClick={onLoginButtonClick}
  //     loginButtonLabel="Log in"
  //   />
  // );
  //
  // return (
  //   <LoginPage
  //     brandImgAlt="PatternFly logo"
  //     backgroundImgSrc="/assets/images/pfbg-icon.svg"
  //     textContent="This is placeholder text only. Use this area to place any information or introductory message about your application that may be relevant to users."
  //     loginTitle="Log in to your account"
  //     loginSubtitle="Enter your single sign-on LDAP credentials."
  //     socialMediaLoginAriaLabel="Log in with social media"
  //   >
  //     {loginForm}
  //   </LoginPage>
  // );

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
      // isLoginButtonDisabled={username === '' || password === ''}
      onLoginButtonClick={onLoginButtonClick}
      loginButtonLabel="Log in"
    />
  );

  // className={'login_component'}
  return (
    <div className={'login_container'}>
      <div>
        <LoginPage
          // className={'login_component'}
          brandImgSrc={logo}
          brandImgAlt="Distributed Dashboard Logo"
          backgroundImgSrc={logo_greyscale}
          footerListItems={footerListItems}
          textContent="Distributed Notebook Cluster | Admin Dashboard & Workload Orchestrator"
          loginTitle="Log in to access the Dashboard"
          loginSubtitle="Enter the configured admin credentials"
        >
          {loginForm}
        </LoginPage>
      </div>
    </div>
  );
};
