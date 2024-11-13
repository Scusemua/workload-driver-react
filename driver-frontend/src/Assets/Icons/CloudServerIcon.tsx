import { SVGIconProps } from '@patternfly/react-icons/dist/esm/createIcon';
import React from 'react';

interface CloudServerIconProps extends SVGIconProps {
    scale?: number;
}

export class CloudServerIcon extends React.Component<CloudServerIconProps> {
    static displayName = 'CloudServerIcon';

    id = `icon-title-cloud-server-icon`;

    render() {
        let scale = 1;
        if (this.props.scale) {
            scale = this.props.scale;
        }

        const { title, className, ...props } = this.props;
        const classes = className ? `pf-v6-svg ${className}` : 'pf-v6-svg';

        const hasTitle = Boolean(title);
        const viewBox = [0, 0, 472.615, 472.615].join(' ');

        return (
            <svg
                className={classes}
                viewBox={viewBox}
                fill="currentColor"
                aria-labelledby={hasTitle ? this.id : undefined}
                aria-hidden={hasTitle ? undefined : true}
                role="img"
                width="1em"
                height="1em"
                transform={`scale(${scale})`}
                {...(props as Omit<React.SVGProps<SVGElement>, 'ref'>)} // Lie.
            >
                {hasTitle && <title id={this.id}>{title}</title>}
                <g>
                    <g>
                        <path
                            d="M405.563,151.927c4.234-13.59,5.908-27.669,5.12-42.143C407.138,49.722,356.234,0.491,297.157,0h-0.886
			c-47.754,0-90.289,30.622-107.028,75.126c-17.526-14.572-40.074-21.563-63.015-19.297c-32.788,3.248-61.243,26.19-72.665,58.485
			c-5.12,14.375-6.4,29.145-3.742,44.012C19.692,173.194,0,204.602,0,239.064c0,46.481,32.614,81.07,78.769,85.387V187.044h315.077
			V324.45c46.155-4.317,78.769-38.906,78.769-85.387C472.615,196.923,444.554,161.477,405.563,151.927z"
                        />
                    </g>
                </g>
                <g>
                    <g>
                        <path
                            d="M374.154,305.231v-98.462H98.462v98.462h49.231v19.692H98.462v98.462h44.308v29.538h-34.462v19.692h256v-19.692h-34.462
			v-29.538h44.308v-98.462h-39.385v-19.692H374.154z M167.385,324.923v-19.692h147.692v19.692H167.385z M295.385,364.275v19.692
			h-19.692v-19.692H295.385z M275.692,246.121h19.692v19.692h-19.692V246.121z M137.846,265.814v-19.692h78.769v19.692H137.846z
			 M137.846,383.968v-19.692h78.769v19.692H137.846z M310.154,452.923H162.462v-29.538h147.692V452.923z M334.769,383.968h-19.692
			v-19.692h19.692V383.968z M315.077,265.814v-19.692h19.692v19.692H315.077z"
                        />
                    </g>
                </g>
            </svg>
        );
    }
}
