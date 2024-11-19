import { GetDefaultSessionFieldValue } from '@Components/Workloads/Constants';
import { SessionTabsDataContext } from '@src/Providers';
import { FormSection, Tab, TabTitleText, Tabs } from '@patternfly/react-core';
import React from 'react';

import { useFieldArray, useFormContext } from 'react-hook-form';

import { v4 as uuidv4 } from 'uuid';
import { SessionConfigurationFormTabContent } from '@Components/Modals';

export interface SessionConfigurationFormProps {
    children?: React.ReactNode;
}

// TODO: Responsive validation not quite working yet.
export const SessionConfigurationForm: React.FunctionComponent<SessionConfigurationFormProps> = () => {
    const {
        control,
        setValue,
        formState: { errors },
    } = useFormContext(); // retrieve all hook methods
    const { append: appendSession, remove: removeSession } = useFieldArray({ name: 'sessions', control });

    const { activeSessionTab, setActiveSessionTab, sessionTabs, addSessionTabs, removeSessionTabs } =
        React.useContext(SessionTabsDataContext);

    const sessionTabComponentRef = React.useRef();
    const firstSessionTabMount = React.useRef<boolean>(true);

    React.useEffect(() => {
        console.log(`Workload template validation errors: ${JSON.stringify(errors)}`);
    }, [errors]);

    const onSessionTabSelect = (tabIndex: number) => {
        setActiveSessionTab(tabIndex);
    };

    const onCloseSessionTab = (_: React.MouseEvent<HTMLElement, MouseEvent>, tabIndex: string | number) => {
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
        setValue('numberOfSessions', sessionTabs.length);

        console.log(`Session Tabs updated (${sessionTabs.length}): ${JSON.stringify(sessionTabs)}`);

        if (firstSessionTabMount.current) {
            firstSessionTabMount.current = false;
            return;
        } else {
            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
            // @ts-expect-error
            const first = sessionTabComponentRef.current?.tabList.current.childNodes[activeSessionTab];
            if (first) {
                first.firstChild.focus();
            }
        }
    }, [sessionTabs, activeSessionTab, setValue]);

    return (
        <FormSection title={`Workload Sessions`} titleElement="h1">
            <Tabs
                activeKey={activeSessionTab}
                onSelect={(_: React.MouseEvent<HTMLElement, MouseEvent>, eventKey: number | string) => {
                    onSessionTabSelect(eventKey as number);
                }}
                // onClose={(_, idx: string | number) => removeSessionTab(idx as number)}
                onClose={onCloseSessionTab}
                onAdd={onAddSessionTab} // addSessionTab
                addButtonAriaLabel="Add Additional Session to Workload"
                role="region"
                // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                // @ts-expect-error
                ref={sessionTabComponentRef}
                aria-label="Session Configuration Tabs"
            >
                {sessionTabs.map((tabName: string, sessionTabIndex: number) => {
                    const defaultSessionId: string = uuidv4();
                    // console.log(`"Passing default session ID ${defaultSessionId} to tab #${sessionTabIndex}`);
                    return (
                        <Tab
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
                        </Tab>
                    );
                })}
            </Tabs>
        </FormSection>
    );
};
