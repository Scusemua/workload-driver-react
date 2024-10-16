import React from 'react';
import { Button, Modal, ModalVariant } from '@patternfly/react-core';

import { ReactSvgPanZoomLoader } from 'react-svg-pan-zoom-loader';
import { INITIAL_VALUE, ReactSVGPanZoom, TOOL_NONE } from 'react-svg-pan-zoom';
import { Workload } from '@app/Data/Workload';

export interface VisualizeWorkloadModalProps {
    children?: React.ReactNode;
    isOpen: boolean;
    onClose: () => void;
    workload: Workload | null;
}

export const VisualizeWorkloadModal: React.FunctionComponent<VisualizeWorkloadModalProps> = (props) => {
    const Viewer = React.useRef(null);
    const [tool, setTool] = React.useState(TOOL_NONE);
    const [value, setValue] = React.useState(INITIAL_VALUE);

    return (
        <Modal
            variant={ModalVariant.small}
            titleIconVariant={'info'}
            aria-label="visualize-workload-modal"
            title={`Inspecting Workload ${props.workload?.name}`}
            isOpen={props.isOpen}
            onClose={props.onClose}
            actions={[
                <Button key="dismiss" variant="primary" onClick={props.onClose}>
                    Dismiss
                </Button>,
            ]}
        >
            <ReactSvgPanZoomLoader
                svgXML={props.workload?.workload_preset.svg_content}
                render={(content) => (
                    <ReactSVGPanZoom
                        width={500}
                        height={500}
                        ref={Viewer}
                        tool={tool}
                        onChangeTool={setTool}
                        value={value}
                        onChangeValue={setValue}
                        onZoom={() => console.log('zoom')}
                        onPan={() => console.log('pan')}
                    >
                        <svg width={500} height={500}>
                            {content}
                        </svg>
                    </ReactSVGPanZoom>
                )}
            />
        </Modal>
    );
};
