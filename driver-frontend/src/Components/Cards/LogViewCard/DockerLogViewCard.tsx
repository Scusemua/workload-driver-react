import { useClusterAge } from '@src/Providers';
import {
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    TextInput,
    TextInputProps,
    Title,
    ToggleGroup,
    ToggleGroupItem,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
    Tooltip,
} from '@patternfly/react-core';
import { MoonIcon, SunIcon } from '@patternfly/react-icons';
import React, { useState } from 'react';

const default_card_height: number = 800;
const min_card_height: number = 100;
const max_card_height: number = 2500;

const darkModeButtonId: string = 'docker-logs-dark-mode-toggle-group-item-button';
const lightModeButtonId: string = 'docker-logs-light-mode-toggle-group-item-button';

export const DockerLogHeightContext = React.createContext(default_card_height);

export const DockerLogViewCard: React.FunctionComponent = () => {
    const [dockerLogHeight, setDockerLogHeight] = useState(default_card_height);
    const [dockerLogHeightString, setDockerLogHeightString] = React.useState(default_card_height.toString());
    const [dockerLogHeightValidated, setDockerLogHeightValidated] =
        React.useState<TextInputProps['validated']>('default');
    const [selectedLogTheme, setSelectedLogTheme] = React.useState(lightModeButtonId);

    const { clusterAge } = useClusterAge();

    const currentTime = React.useRef<number>(Date.now());

    const getGatewayLogsUrl = () => {
        if (selectedLogTheme == darkModeButtonId) {
            return `http://localhost:3000/d-solo/ddx4gnyl0cmbka/distributed-cluster-primary-metrics?orgId=1&refresh=5s&from=${(clusterAge > 0 ? clusterAge : Date.now()) - 300000}&to=${currentTime.current ? currentTime.current : Date.now()}&panelId=140&theme=dark`;
        } else {
            return `http://localhost:3000/d-solo/ddx4gnyl0cmbka/distributed-cluster-primary-metrics?orgId=1&refresh=5s&from=${(clusterAge > 0 ? clusterAge : Date.now()) - 300000}&to=${currentTime.current ? currentTime.current : Date.now()}&panelId=140&theme=light`;
        }
    };

    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
    // @ts-expect-error
    const handleThemeToggleClick = (event: MouseEvent | React.MouseEvent<any, MouseEvent> | KeyboardEvent<Element>) => {
        const id = event.currentTarget.id;
        console.log(`Setting log theme to ${id}`);
        setSelectedLogTheme(id);
    };

    const onHeightTextboxChanged = (_event: React.FormEvent<HTMLInputElement>, value: string) => {
        setDockerLogHeightString(value);

        if (value == '') {
            setDockerLogHeight(default_card_height);
            return;
        }

        const height: number = Number.parseInt(value);
        if (Number.isNaN(height)) {
            setDockerLogHeightValidated('error');
            return;
        }

        if (height < min_card_height) {
            setDockerLogHeightValidated('error');
            return;
        }

        if (height > max_card_height) {
            setDockerLogHeightValidated('error');
            return;
        }

        setDockerLogHeightValidated('default');
        setDockerLogHeight(height);
    };

    const cardHeaderActions = (
        <Toolbar>
            <ToolbarContent>
                <React.Fragment>
                    <ToolbarGroup>
                        <ToolbarItem>
                            <Tooltip
                                id="log-card-height-text-input-tooltip"
                                aria-label="log-card-height-text-input-tooltip"
                                exitDelay={75}
                                content={<div>Specify the height of the &quot;Logs&quot; card.</div>}
                            >
                                <TextInput
                                    aria-label="log-card-height-text-input"
                                    id="log-card-height-text-input"
                                    placeholder={dockerLogHeight.toString()}
                                    value={dockerLogHeightString}
                                    type="number"
                                    validated={dockerLogHeightValidated}
                                    onChange={onHeightTextboxChanged}
                                />
                            </Tooltip>
                        </ToolbarItem>
                        <ToolbarItem>
                            <ToggleGroup>
                                <ToggleGroupItem
                                    aria-label={'Toggle Docker Logs light mode'}
                                    id={lightModeButtonId}
                                    buttonId={lightModeButtonId}
                                    icon={<SunIcon />}
                                    onChange={handleThemeToggleClick}
                                    isSelected={selectedLogTheme === lightModeButtonId}
                                />
                                <ToggleGroupItem
                                    aria-label={'Toggle Docker Logs dark mode'}
                                    id={darkModeButtonId}
                                    buttonId={darkModeButtonId}
                                    icon={<MoonIcon />}
                                    onChange={handleThemeToggleClick}
                                    isSelected={selectedLogTheme === darkModeButtonId}
                                />
                            </ToggleGroup>
                        </ToolbarItem>
                    </ToolbarGroup>
                </React.Fragment>
            </ToolbarContent>
        </Toolbar>
    );

    return (
        <DockerLogHeightContext.Provider value={dockerLogHeight}>
            <Card isFullHeight style={{ height: dockerLogHeight, overflow: 'auto' }}>
                <CardHeader actions={{ actions: cardHeaderActions, hasNoOffset: false }}>
                    <CardTitle>
                        <Title headingLevel="h1" size="xl">
                            Logs
                        </Title>
                    </CardTitle>
                </CardHeader>
                <CardBody style={{ height: '400px', overflow: 'auto' }}>
                    <iframe src={getGatewayLogsUrl()} style={{ height: '100%', width: '100%' }} />
                </CardBody>
            </Card>
        </DockerLogHeightContext.Provider>
    );
};
