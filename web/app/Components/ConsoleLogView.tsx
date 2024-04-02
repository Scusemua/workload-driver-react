import React, { useEffect, useRef } from 'react';
import { Panel, PanelMain, PanelMainBody } from '@patternfly/react-core';
import { Console, Hook, Unhook } from 'console-feed';
import { Message } from 'console-feed/lib/definitions/Console';
import { Message as MessageComponent } from 'console-feed/lib/definitions/Component';

export interface ConsoleLogViewProps {
    children?: React.ReactNode;
}

export const ConsoleLogViewComponent: React.FunctionComponent<ConsoleLogViewProps> = () => {
    const logs = useRef<MessageComponent[]>([]);

    useEffect(() => {
        const hookedConsole = Hook(
            window.console,
            (log: Message) => {
                logs.current = [...logs.current, log as MessageComponent];
            },
            false,
        );
        return () => {
            Unhook(hookedConsole);
        };
    }, []);

    return (
        <Panel isScrollable variant="bordered">
            <PanelMain maxHeight={'450px'}>
                <PanelMainBody>
                    <Console logs={logs.current} variant="dark" />
                </PanelMainBody>
            </PanelMain>
        </Panel>
    );
};
