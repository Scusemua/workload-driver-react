import React from 'react';
import {
    FormSection,
    Tabs,
    Tab,
    TabTitleText,
    TabAction,
} from '@patternfly/react-core';

import { v4 as uuidv4 } from 'uuid';

import { useFieldArray, useFormContext } from 'react-hook-form';
import { SessionConfigurationFormTabContent } from './SessionConfigurationFormTabContent';
import { TimesIcon } from '@patternfly/react-icons';

export interface SessionConfigurationFormProps {
    children?: React.ReactNode;
}

// TODO: Responsive validation not quite working yet.
export const SessionConfigurationForm: React.FunctionComponent<SessionConfigurationFormProps> = (props) => {
    const { control, formState: { errors } } = useFormContext() // retrieve all hook methods
    const { fields, append, remove } = useFieldArray({ name: "sessions", control });

    const [activeSessionTab, setActiveSessionTab] = React.useState<number>(0);
    const [sessionTabs, setSessionTabs] = React.useState<string[]>(['Session 1']);
    const [newSessionTabNumber, setNewSessionTabNumber] = React.useState<number>(2);
    const sessionTabComponentRef = React.useRef<any>();
    const firstSessionTabMount = React.useRef<boolean>(true);

    React.useEffect(() => {
        console.log(errors);
    }, [errors]);

    const onSessionTabSelect = (
        tabIndex: number
    ) => {
        setActiveSessionTab(tabIndex);
    };

    const onCloseSessionTab = (event: any, tabIndex: string | number) => {
        const tabIndexNum = tabIndex as number;
        let nextTabIndex = activeSessionTab;
        if (tabIndexNum < activeSessionTab) {
            // if a preceding tab is closing, keep focus on the new index of the current tab
            nextTabIndex = activeSessionTab - 1 > 0 ? activeSessionTab - 1 : 0;
        } else if (activeSessionTab === sessionTabs.length - 1) {
            // if the closing tab is the last tab, focus the preceding tab
            nextTabIndex = sessionTabs.length - 2 > 0 ? sessionTabs.length - 2 : 0;
        }
        setActiveSessionTab(nextTabIndex);
        setSessionTabs(sessionTabs.filter((_, index) => index !== tabIndex));

        remove(tabIndex as number);
    };

    const onAddSessionTab = () => {
        setSessionTabs([...sessionTabs, `Session ${newSessionTabNumber}`]);
        setActiveSessionTab(sessionTabs.length);
        setNewSessionTabNumber(newSessionTabNumber + 1);

        append({
            id: uuidv4(),
            start_tick: 1,
            stop_tick: 6,
            training_start_tick: 2,
            training_duration_ticks: 2,
            cpu_percent_util: 10,
            mem_usage_gb_util: 0.25,
            num_gpus: 1,
            gpu_utilizations: [{
                utilization: 100,
            }]
        })
    };

    React.useEffect(() => {
        if (firstSessionTabMount.current) {
            firstSessionTabMount.current = false;
            return;
        } else {
            const first = sessionTabComponentRef.current?.tabList.current.childNodes[activeSessionTab];
            first && first.firstChild.focus();
        }

        // const newVal: number = sessionTabs.length;
        // const oldVal: number = fields.length;
        // console.log(`Old (fields.length): ${oldVal}, New (sessionTabs.length): ${newVal}`);
        // if (newVal > oldVal) {
        //     // Append sessions to field array
        //     for (let i = oldVal; i < newVal; i++) {
        //         console.log(`Adding new session field. fields.length pre-add: ${fields.length}. i: ${i}, oldVal: ${oldVal}, newVal: ${newVal}.`)
        //         append({});
        //         console.log(`Added new session field. fields.length post-add: ${fields.length}. i: ${i}, oldVal: ${oldVal}, newVal: ${newVal}.`)
        //     }
        // } else {
        //     // Remove sessions from field array
        //     for (let i = oldVal; i > newVal; i--) {
        //         console.log(`Removing session field. fields.length pre-removal: ${fields.length}`)
        //         remove(i - 1);
        //         console.log(`Removed session field. fields.length post-removal: ${fields.length}`)
        //     }
        // }
    }, [sessionTabs]);


    return (
        <FormSection title={`Workload Sessions (${sessionTabs.length})`} titleElement='h1' >
            <Tabs
                isFilled
                activeKey={activeSessionTab}
                onSelect={(_: React.MouseEvent<HTMLElement, MouseEvent>, eventKey: number | string) => { onSessionTabSelect(eventKey as number) }}
                isBox={true}
                onClose={onCloseSessionTab}
                onAdd={onAddSessionTab}
                addButtonAriaLabel='Add Additional Session to Workload'
                role='region'
                ref={sessionTabComponentRef}
                aria-label="Session Configuration Tabs"
            >
                {sessionTabs.map((tabName: string, tabIndex: number) => {
                    const tabRef = React.createRef<HTMLElement>();
                    
                    return (<Tab
                        key={tabIndex}
                        eventKey={tabIndex}
                        aria-label={`${tabName} Tab`}
                        title={<TabTitleText>{tabName}</TabTitleText>}
                        closeButtonAriaLabel={`Close ${tabName} Tab`}
                        isCloseDisabled={sessionTabs.length == 1}
                    >
                        <SessionConfigurationFormTabContent tabIndex={tabIndex} defaultSessionId={uuidv4()}/>
                    </Tab>)
                })}
            </Tabs>
        </FormSection>
    )
}