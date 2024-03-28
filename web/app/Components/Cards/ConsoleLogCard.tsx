/* eslint-disable camelcase */
import React, { useState, useEffect } from 'react';
import { Card, CardHeader, CardTitle, CardBody, Panel, PanelMain, PanelMainBody, Title } from '@patternfly/react-core';

import { Console, Hook, Unhook } from 'console-feed';

export const ConsoleLogCard: React.FunctionComponent = () => {
    const [logs, setLogs] = useState([]);
    const [maxHeight, setMaxHeight] = useState('450px');

    useEffect(() => {
        const hookedConsole = Hook(window.console, (log) => setLogs((currLogs) => [...currLogs, log]), false);
        return () => Unhook(hookedConsole);
    }, []);

    return (
        <Card isFullHeight id="console-log-view-card">
            <CardHeader>
                <CardTitle>
                    <Title headingLevel="h1" size="xl">
                        Console Output
                    </Title>
                </CardTitle>
            </CardHeader>
            <CardBody>
                <Panel isScrollable>
                    <PanelMain maxHeight={maxHeight}>
                        <PanelMainBody>
                            <Console logs={logs} variant="dark" />
                        </PanelMainBody>
                    </PanelMain>
                </Panel>
            </CardBody>
        </Card>
    );
};
