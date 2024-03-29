/* eslint-disable camelcase */
import React, { useEffect, useState } from 'react';
import { Card, CardBody, CardHeader, CardTitle, Panel, PanelMain, PanelMainBody, Title } from '@patternfly/react-core';

import { Console, Hook, Unhook } from 'console-feed';
import { Message } from 'console-feed/lib/definitions/Console';
import { Message as MessageComponent } from 'console-feed/lib/definitions/Component';

export const ConsoleLogCard: React.FunctionComponent = () => {
    const [logs, setLogs] = useState<MessageComponent[]>([]);

    useEffect(() => {
        const hookedConsole = Hook(
            window.console,
            (log: Message) => setLogs((currLogs: MessageComponent[]) => [...currLogs, log as MessageComponent]),
            false,
        );
        return () => {
            Unhook(hookedConsole);
        };
    }, []);

    return (
        <Card isRounded isFullHeight id="console-log-view-card">
            <CardHeader>
                <CardTitle>
                    <Title headingLevel="h1" size="xl">
                        Logs
                    </Title>
                </CardTitle>
            </CardHeader>
            <CardBody>
                <Panel isScrollable>
                    <PanelMain maxHeight={'450px'}>
                        <PanelMainBody>
                            <Console logs={logs} variant="dark" />
                        </PanelMainBody>
                    </PanelMain>
                </Panel>
            </CardBody>
        </Card>
    );
};
