import React, { useEffect, useRef } from 'react';
import { Panel, PanelMain, PanelMainBody } from '@patternfly/react-core';
import { Console, Hook, Unhook } from 'console-feed';
import { Message } from 'console-feed/lib/definitions/Console';
import { Message as MessageComponent } from 'console-feed/lib/definitions/Component';

export interface BrowserDebugConsoleLogViewProps {
    children?: React.ReactNode;
    height: number;
}

export const BrowserDebugConsoleLogView: React.FunctionComponent<BrowserDebugConsoleLogViewProps> = (
    props: BrowserDebugConsoleLogViewProps,
) => {
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
        <Panel isScrollable>
            <PanelMain maxHeight={`${props.height.toString()}px`}>
                <PanelMainBody>
                    <Console logs={logs.current} variant="dark" />
                </PanelMainBody>
            </PanelMain>
        </Panel>
    );
};
