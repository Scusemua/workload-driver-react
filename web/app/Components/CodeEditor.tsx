import React from 'react';
import { ChangeHandler, CodeEditor, CodeEditorControl, Language } from '@patternfly/react-code-editor';
import { Button, Chip, Grid, GridItem, Switch } from '@patternfly/react-core';
import { editor } from 'monaco-editor/esm/vs/editor/editor.api';
import { CodeIcon } from '@patternfly/react-icons';
import { Monaco } from '@monaco-editor/react';

export interface CodeEditorComponent {
    children?: React.ReactNode;
    onChange?: ChangeHandler;
}

export const CodeEditorComponent: React.FunctionComponent<CodeEditorComponent> = (props) => {
    const [isDarkMode, setIsDarkMode] = React.useState(false);
    const [code, setCode] = React.useState('');

    const onEditorDidMount = (editor: editor.IStandaloneCodeEditor, monaco: Monaco) => {
        editor.layout();
        editor.focus();
        monaco.editor.getModels()[0].updateOptions({ tabSize: 5 });
    };

    const shortcuts = [
        {
            keys: ['Opt', 'F1'],
            description: 'Accessibility helps',
        },
        {
            keys: ['F1'],
            description: 'View all editor shortcuts',
        },
        {
            keys: ['Ctrl', 'Space'],
            description: 'Activate auto complete',
        },
        {
            keys: ['Cmd', 'S'],
            description: 'Save',
        },
    ];
    const shortcutsPopoverProps = {
        bodyContent: (
            <Grid span={6} hasGutter key="grid">
                {shortcuts.map((shortcut, index) => (
                    <React.Fragment key={index}>
                        <GridItem style={{ textAlign: 'right', marginRight: '1em' }}>
                            {shortcut.keys
                                .map((key) => (
                                    <Chip key={key} isReadOnly>
                                        {key}
                                    </Chip>
                                ))
                                .reduce((prev, curr) => (
                                    <>{[prev, ' + ', curr]}</>
                                ))}
                        </GridItem>
                        <GridItem>{shortcut.description}</GridItem>
                    </React.Fragment>
                ))}
            </Grid>
        ),
        'aria-label': 'Shortcuts',
    };

    const darkLightThemeSwitch = (
        // <CodeEditorControl
        //     aria-label={'Toggle darkmode' + ((isDarkMode && ' off') || ' on')}
        //     tooltipProps={{
        //         content: 'Toggle darkmode' + ((isDarkMode && ' off') || ' on'),
        //     }}
        //     onClick={() => {
        //         setIsDarkMode(!isDarkMode);
        //     }}
        //     icon={<LightbulbIcon />}
        // />
        <div>
            <Button
                variant="link"
                onClick={(event: React.MouseEvent<HTMLButtonElement, MouseEvent>) => {
                    event.stopPropagation();
                }}
                onMouseDown={(event: React.MouseEvent<HTMLButtonElement, MouseEvent>) => {
                    event.preventDefault();
                }}
            >
                <Switch
                    id="darkmode-switch"
                    aria-label="darkmode-switch"
                    label="Dark Theme"
                    labelOff="Light Theme"
                    isChecked={isDarkMode}
                    onChange={(_event: React.FormEvent<HTMLInputElement>, checked: boolean) => {
                        setIsDarkMode(checked);
                    }}
                />
            </Button>
        </div>
    );

    const defaultCodeTemplate1 = (
        <CodeEditorControl
            icon={<CodeIcon />}
            aria-label="Execute code"
            tooltipProps={{ content: 'Sample Code #1' }}
            onClick={() => {
                setCode(
                    `a = 15
print("a = %d" % a)`,
                );
            }}
        />
    );

    const defaultCodeTemplate2 = (
        <CodeEditorControl
            icon={<CodeIcon />}
            aria-label="Execute code"
            tooltipProps={{ content: 'Sample Code #2' }}
            onClick={() => {
                setCode(
                    `b = a * 2
print("a = %d, b = %d" % (a, b))`,
                );
            }}
        />
    );

    return (
        <CodeEditor
            isDarkTheme={isDarkMode}
            shortcutsPopoverProps={shortcutsPopoverProps}
            customControls={[defaultCodeTemplate1, defaultCodeTemplate2, darkLightThemeSwitch]}
            isLanguageLabelVisible
            isUploadEnabled
            isDownloadEnabled
            isCopyEnabled
            code={code}
            onChange={(value: string, event: editor.IModelContentChangedEvent) => {
                setCode(value);
                if (props.onChange) {
                    props.onChange(value, event);
                }
            }}
            language={Language.python}
            onEditorDidMount={onEditorDidMount}
            height="400px"
        />
    );
};
