import React from 'react';
import { ChangeHandler, CodeEditor, Language } from '@patternfly/react-code-editor';
import { Button, Chip, Grid, GridItem, Switch } from '@patternfly/react-core';

export interface CodeEditorComponent {
    children?: React.ReactNode;
    onChange?: ChangeHandler;
}

export const CodeEditorComponent: React.FunctionComponent<CodeEditorComponent> = (props) => {
    const [isDarkMode, setIsDarkMode] = React.useState(false);

    const onEditorDidMount = (editor, monaco) => {
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

    const customControl = (
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

    return (
        <CodeEditor
            isDarkTheme={isDarkMode}
            shortcutsPopoverProps={shortcutsPopoverProps}
            customControls={customControl}
            isLanguageLabelVisible
            isUploadEnabled
            isDownloadEnabled
            isCopyEnabled
            onChange={props.onChange}
            language={Language.python}
            onEditorDidMount={onEditorDidMount}
            height="400px"
        />
    );
};
