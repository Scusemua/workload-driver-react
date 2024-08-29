import {SessionTabsDataContext} from "@app/Providers";
import {FormSection, Tab, TabTitleText, Tabs,} from '@patternfly/react-core';
import React from 'react';

import {useFieldArray, useFormContext} from 'react-hook-form';

import {v4 as uuidv4} from 'uuid';
import {GetDefaultSessionFieldValue} from './Constants';
import {SessionConfigurationFormTabContent} from '@components/Modals';

export interface SessionConfigurationFormProps {
  children?: React.ReactNode;
}

// TODO: Responsive validation not quite working yet.
export const SessionConfigurationForm: React.FunctionComponent<SessionConfigurationFormProps> = () => {
  const {control, setValue, formState: {errors}} = useFormContext() // retrieve all hook methods
  const {append: appendSession, remove: removeSession} = useFieldArray({name: "sessions", control});

  // const [activeSessionTab, setActiveSessionTab] = React.useState<number>(0);
  // const [sessionTabs, setSessionTabs] = React.useState<string[]>(['Session 1']);
  // const [newSessionTabNumber, setNewSessionTabNumber] = React.useState<number>(2);

  const {
    activeSessionTab,
    setActiveSessionTab,
    sessionTabs,
    addSessionTabs,
    removeSessionTabs
  } = React.useContext(SessionTabsDataContext);

  const sessionTabComponentRef = React.useRef();
  const firstSessionTabMount = React.useRef<boolean>(true);

  React.useEffect(() => {
    console.log(errors);
  }, [errors]);

  const onSessionTabSelect = (
    tabIndex: number
  ) => {
    setActiveSessionTab(tabIndex);
  };

  const onCloseSessionTab = (_: React.MouseEvent<HTMLElement, MouseEvent>, tabIndex: string | number) => {
    // const tabIndexNum = tabIndex as number;
    // let nextTabIndex = activeSessionTab;
    // if (tabIndexNum < activeSessionTab) {
    //   // if a preceding tab is closing, keep focus on the new index of the current tab
    //   nextTabIndex = activeSessionTab - 1 > 0 ? activeSessionTab - 1 : 0;
    // } else if (activeSessionTab === sessionTabs.length - 1) {
    //   // if the closing tab is the last tab, focus the preceding tab
    //   nextTabIndex = sessionTabs.length - 2 > 0 ? sessionTabs.length - 2 : 0;
    // }
    // setActiveSessionTab(nextTabIndex);
    // setSessionTabs(sessionTabs.filter((_, index) => index !== tabIndex));
    removeSessionTabs([tabIndex as number]);

    removeSession(tabIndex as number);
  };

  const onAddSessionTab = () => {
    // setSessionTabs([...sessionTabs, `Session ${newSessionTabNumber}`]);
    // setActiveSessionTab(sessionTabs.length);
    // setNewSessionTabNumber(newSessionTabNumber + 1);
    addSessionTabs(1);
    appendSession(GetDefaultSessionFieldValue());
  };

  React.useEffect(() => {
    setValue("numberOfSessions", sessionTabs.length);

    console.log(`Session Tabs updated (${sessionTabs.length}): ${JSON.stringify(sessionTabs)}`)

    // if (sessionTabs.length > sessionFields.length) {
    //   const diff: number = sessionTabs.length - sessionFields.length;
    //
    //   console.log(`Need to add ${diff} session fields(s).`)
    //
    //   for (let i: number = 0; i < diff; i++) {
    //     console.log("Adding session field...");
    //     appendSession(GetDefaultSessionFieldValue());
    //   }
    // } else if (sessionTabs.length < sessionFields.length) {
    //   const diff: number = sessionFields.length - sessionTabs.length;
    //
    //   console.log(`Need to remove ${diff} session fields(s).`)
    //
    //   let idx: number = sessionTabs.length - 1;
    //   for (let i: number = 0; i < diff; i++) {
    //     console.log(`Removing session field #${idx}...`);
    //     removeSession(idx);
    //     idx = idx - 1;
    //   }
    // }

    if (firstSessionTabMount.current) {
      firstSessionTabMount.current = false;
      return;
    } else {
      const first = sessionTabComponentRef.current?.tabList.current.childNodes[activeSessionTab];
      first && first.firstChild.focus();
    }
  }, [sessionTabs, activeSessionTab, setValue]);

  return (
    <FormSection title={`Workload Sessions`} titleElement='h1'>
      <Tabs
        activeKey={activeSessionTab}
        onSelect={(_: React.MouseEvent<HTMLElement, MouseEvent>, eventKey: number | string) => {
          onSessionTabSelect(eventKey as number)
        }}
        // onClose={(_, idx: string | number) => removeSessionTab(idx as number)}
        onClose={onCloseSessionTab}
        onAdd={onAddSessionTab} // addSessionTab
        addButtonAriaLabel='Add Additional Session to Workload'
        role='region'
        ref={sessionTabComponentRef}
        aria-label="Session Configuration Tabs"
      >
        {sessionTabs.map((tabName: string, sessionTabIndex: number) => {
          const defaultSessionId: string = uuidv4();
          // console.log(`"Passing default session ID ${defaultSessionId} to tab #${sessionTabIndex}`);
          return (<Tab
            id={`session${sessionTabIndex}-tab-id`}
            key={`session${sessionTabIndex}-tab-key`}
            eventKey={sessionTabIndex}
            aria-label={`${tabName} Tab`}
            title={<TabTitleText>{tabName}</TabTitleText>}
            closeButtonAriaLabel={`Close ${tabName} Tab`}
            isCloseDisabled={sessionTabs.length == 1}
          >
            <SessionConfigurationFormTabContent
              sessionIndex={sessionTabIndex}
              defaultSessionId={defaultSessionId}
            />
          </Tab>)
        })}
      </Tabs>
    </FormSection>
  )
}
