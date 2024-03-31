import { SVGIconProps } from '@patternfly/react-icons/dist/esm/createIcon';
import React from 'react';

interface GpuIconAltProps extends SVGIconProps {
    scale?: number;
}

export class GpuIconAlt extends React.Component<GpuIconAltProps> {
    static displayName = 'GpuIconAlt';

    id = `icon-title-gpu-alt-icon`;

    render() {
        let scale = 1;
        if (this.props.scale) {
            scale = this.props.scale;
        }

        const { title, className, ...props } = this.props;
        const classes = className ? `pf-v5-svg ${className}` : 'pf-v5-svg';

        const hasTitle = Boolean(title);
        const viewBox = [0, 25, 832, 1024].join(' ');

        return (
            <svg
                className={classes}
                viewBox={viewBox}
                aria-labelledby={hasTitle ? this.id : undefined}
                aria-hidden={hasTitle ? undefined : true}
                role="img"
                width="1em"
                height="1em"
                transform={`scale(${scale})`}
                {...(props as Omit<React.SVGProps<SVGElement>, 'ref'>)} // Lie.
            >
                {hasTitle && <title id={this.id}>{title}</title>}
                <path
                    d="M963.718737 567.969684c0-31.636211-19.240421-68.419368-42.954105-82.108631L181.490526 59.041684c-23.713684-13.689263-42.954105 0.862316-42.954105 32.498527v362.819368c0 31.636211 19.240421 68.419368 42.954105 82.108632l739.274106 426.819368c23.713684 13.689263 42.954105-0.889263 42.954105-32.498526V567.969684z"
                    fill="#39549F"
                />
                <path
                    d="M66.236632 105.094737l88.495157-49.017263 21.234527 39.801263-37.429895 50.58021L66.236632 105.094737zM865.253053 1013.786947l85.315368-49.259789-77.392842-49.744842-7.922526 99.004631z"
                    fill="#39549F"
                />
                <path
                    d="M880.801684 615.828211c0-31.609263-19.267368-68.392421-42.981052-82.081685L98.573474 106.927158C74.859789 93.237895 55.592421 107.789474 55.592421 139.398737v362.846316c0 31.609263 19.267368 68.392421 42.981053 82.081684l739.247158 426.819368c23.713684 13.689263 42.981053-0.862316 42.981052-32.471579V615.828211z"
                    fill="#18171C"
                />
                <path
                    d="M467.348211 558.268632s-122.556632-234.765474-177.906527-316.254316c-5.308632-7.814737-21.207579-34.654316-28.294737-41.498948-8.003368-7.706947-28.672-16.545684-35.058526-20.237473L96.848842 105.660632C73.135158 91.971368 53.894737 106.522947 53.894737 138.159158v362.819368c0 31.636211 19.240421 68.419368 42.954105 82.108632l141.985684 81.946947c7.248842 4.203789 43.52 5.820632 54.406737 2.425263 34.438737-10.725053 174.106947-87.471158 174.106948-87.471157v-21.719579z m-30.989474-17.030737s-104.178526-199.572211-151.228632-268.853895c-4.527158-6.656-18.027789-29.453474-24.064-35.301053-6.790737-6.548211-24.387368-14.066526-29.803789-17.192421L121.397895 156.456421c-20.156632-11.641263-36.513684 0.727579-36.513684 27.621053v308.466526c0 26.866526 16.357053 58.152421 36.513684 69.793684l120.697263 69.685895c6.170947 3.557053 36.998737 4.931368 46.241684 2.048 29.291789-9.108211 148.021895-74.347789 148.021895-74.34779v-18.485894z"
                    fill="#6D8ACA"
                />
                <path
                    d="M467.348211 558.538105s122.556632 234.765474 177.906526 316.254316c5.308632 7.814737 21.207579 34.654316 28.294737 41.525895 8.003368 7.706947 28.672 16.545684 35.058526 20.237473l129.212632 74.590316c23.713684 13.689263 42.981053-0.862316 42.981052-32.471579V615.828211c0-31.609263-19.267368-68.392421-42.981052-82.081685l-141.958737-81.973894c-7.248842-4.203789-43.52-5.820632-54.406737-2.425264-34.438737 10.725053-174.106947 87.471158-174.106947 87.471158v21.719579z m30.989473 17.030737s104.178526 199.572211 151.228632 268.853895c4.527158 6.656 18.027789 29.480421 24.064 35.301052 6.790737 6.548211 24.387368 14.066526 29.803789 17.219369l109.864421 63.407158c20.156632 11.641263 36.513684-0.727579 36.513685-27.621053V624.289684c0-26.893474-16.357053-58.152421-36.513685-69.793684l-120.697263-69.685895c-6.170947-3.584-36.998737-4.958316-46.241684-2.074947-29.291789 9.135158-148.021895 74.374737-148.021895 74.374737v18.458947z"
                    fill="#6D8ACA"
                />
                <path d="M436.358737 541.237895s-104.178526-199.572211-151.228632-268.853895c-4.527158-6.656-18.027789-29.453474-24.064-35.301053-6.790737-6.548211-24.387368-14.066526-29.803789-17.192421L121.397895 156.456421c-20.156632-11.641263-36.513684 0.727579-36.513684 27.621053v308.466526c0 26.866526 16.357053 58.152421 36.513684 69.793684l120.697263 69.685895c6.170947 3.557053 36.998737 4.931368 46.241684 2.048 29.291789-9.108211 148.021895-74.347789 148.021895-74.34779v-18.485894zM498.337684 575.568842s104.178526 199.572211 151.228632 268.853895c4.527158 6.656 18.027789 29.480421 24.064 35.301052 6.790737 6.548211 24.387368 14.066526 29.803789 17.219369l109.864421 63.407158c20.156632 11.641263 36.513684-0.727579 36.513685-27.621053V624.289684c0-26.893474-16.357053-58.152421-36.513685-69.793684l-120.697263-69.685895c-6.170947-3.584-36.998737-4.958316-46.241684-2.074947-29.291789 9.135158-148.021895 74.374737-148.021895 74.374737v18.458947z" />
                <path
                    d="M476.725895 531.698526l-34.115369 19.698527 18.458948 32.013473 28.672-16.545684-13.015579-35.166316z"
                    fill="#6D8ACA"
                />
                <path
                    d="M164.136421 410.947368l-10.428632-6.009263c-1.131789-2.209684-1.670737-6.925474-1.670736-14.201263v-0.080842l0.053894-3.233684 0.10779-6.925474c0-1.859368-1.077895-3.422316-3.260632-4.661895v26.273685l-10.401684-6.009264v-54.784l12.853895 7.410527a26.085053 26.085053 0 0 1 7.653052 6.763789 14.713263 14.713263 0 0 1 3.260632 9.242948v6.575157c0 4.257684-2.128842 5.901474-6.440421 4.904422 4.311579 3.691789 6.494316 7.949474 6.494316 12.8l-0.161684 5.901473c0 8.784842 0.646737 14.120421 1.94021 16.033684z m-11.991579-42.819368v-9.054316c0-1.913263-1.050947-3.476211-3.206737-4.715789v14.821052c2.155789 1.239579 3.206737 0.889263 3.206737-1.050947zM187.284211 376.616421l-5.820632-3.368421v47.642947l-10.401684-6.009263v-47.642947l-5.847579-3.368421v-7.141053l22.069895 12.746105v7.141053zM218.758737 442.421895l-11.183158-6.467369-4.769684-20.37221-4.446316 15.036631-9.970526-5.739789 8.192-26.408421-6.063158-27.162948 10.024421 5.793685 3.691789 18.027789 4.149895-13.500631 9.350737 5.416421-7.114105 21.261473 8.138105 34.115369zM231.936 427.061895l9.566316 5.52421v12.988632c0 1.347368 0.619789 2.371368 1.886316 3.098947 1.239579 0.727579 1.886316 0.431158 1.886315-0.91621v-12.692211a5.982316 5.982316 0 0 0-0.943158-3.503158 11.425684 11.425684 0 0 0-3.530105-2.829473v-6.170948c2.964211 1.697684 4.473263 1.509053 4.473263-0.646737v-9.701052c0-1.104842-0.646737-1.994105-1.886315-2.721684-1.320421-0.754526-1.967158-0.592842-1.967158 0.512v10.078315l-9.216-5.335579v-10.159158c0-6.467368 3.853474-7.464421 11.614315-2.991157 3.098947 1.805474 5.820632 4.149895 8.138106 7.033263a14.659368 14.659368 0 0 1 3.47621 9.377684v7.518316c0 4.392421-1.805474 6.278737-5.416421 5.712842a17.246316 17.246316 0 0 1 5.578105 13.365895v8.138105c0 7.626105-3.934316 9.162105-11.749052 4.661895-7.949474-4.608-11.910737-10.347789-11.910737-17.273264v-13.069473zM257.428211 455.383579V420.109474c0-7.087158 4.096-8.272842 12.288-3.557053 8.030316 4.634947 12.045474 10.482526 12.045473 17.542737v35.247158c0 3.152842-1.131789 4.958316-3.395368 5.389473-2.263579 0.431158-5.173895-0.377263-8.730948-2.45221a28.941474 28.941474 0 0 1-8.838736-7.653053 15.333053 15.333053 0 0 1-3.368421-9.242947z m13.904842 8.59621v-37.322105c0-1.104842-0.565895-1.967158-1.697685-2.613895-1.185684-0.700632-1.778526-0.485053-1.778526 0.592843v37.349052c0 1.077895 0.592842 1.967158 1.778526 2.66779 1.131789 0.646737 1.697684 0.431158 1.697685-0.673685zM283.755789 457.054316l10.132211 5.847579v13.958737c0 1.185684 0.592842 2.128842 1.751579 2.802526 1.131789 0.646737 1.697684 0.377263 1.697684-0.808421v-16.276211c-0.754526 0.485053-2.128842 0.134737-4.176842-1.050947a20.938105 20.938105 0 0 1-6.790737-6.224842 13.850947 13.850947 0 0 1-2.613895-7.949474v-12.773052c0-3.179789 1.077895-5.012211 3.260632-5.470316 2.182737-0.458105 5.039158 0.323368 8.542316 2.371368 3.584 2.048 6.467368 4.581053 8.650105 7.545263a15.36 15.36 0 0 1 3.287579 9.189053v35.301053c0 7.087158-3.988211 8.326737-11.991579 3.691789-7.841684-4.527158-11.749053-10.293895-11.749053-17.327158v-12.826947z m13.581474-2.15579v-13.150315a3.072 3.072 0 0 0-1.697684-2.775579c-1.158737-0.673684-1.751579-0.404211-1.751579 0.781473v13.150316c0 1.185684 0.592842 2.128842 1.751579 2.802526 1.131789 0.646737 1.697684 0.377263 1.697684-0.808421zM309.409684 485.402947v-35.274105c0-7.114105 4.096-8.299789 12.288-3.557053 8.030316 4.634947 12.045474 10.482526 12.045474 17.51579v35.274105c0 3.152842-1.131789 4.958316-3.395369 5.362527-2.263579 0.431158-5.173895-0.377263-8.730947-2.425264a28.348632 28.348632 0 0 1-8.838737-7.68 15.252211 15.252211 0 0 1-3.368421-9.216z m13.904842 8.596211v-37.322105c0-1.104842-0.565895-1.967158-1.697684-2.640842-1.185684-0.673684-1.778526-0.458105-1.778526 0.619789v37.322105c0 1.104842 0.592842 1.994105 1.778526 2.694737 1.131789 0.646737 1.697684 0.404211 1.697684-0.673684z"
                    fill="#A8A8A8"
                />
                <path
                    d="M704.727579 535.498105c69.443368 40.097684 125.817263 137.781895 125.817263 217.977263s-56.373895 112.747789-125.817263 72.650106c-69.470316-40.097684-125.844211-137.781895-125.844211-217.977263 0-80.168421 56.373895-112.747789 125.844211-72.650106z"
                    fill="#3B57A6"
                />
                <path
                    d="M693.948632 541.722947c69.443368 40.097684 125.844211 137.754947 125.84421 217.950316 0 80.195368-56.400842 112.747789-125.84421 72.650105s-125.844211-137.754947-125.844211-217.950315c0-80.195368 56.400842-112.747789 125.844211-72.650106z"
                    fill="#4762AF"
                />
                <path
                    d="M693.948632 645.712842c19.752421 11.398737 35.786105 39.181474 35.786105 61.978947 0 22.797474-16.033684 32.040421-35.786105 20.641685-19.752421-11.398737-35.786105-39.154526-35.786106-61.952 0-22.824421 16.033684-32.067368 35.786106-20.668632z"
                    fill="#6D8ACA"
                />
                <path
                    d="M693.948632 655.467789c15.090526 8.704 27.324632 29.911579 27.324631 47.346527 0 17.408-12.234105 24.468211-27.324631 15.76421-15.090526-8.704-27.324632-29.911579-27.324632-47.319579 0-17.434947 12.234105-24.495158 27.324632-15.791158z"
                    fill="#39549F"
                />
                <path
                    d="M676.756211 535.902316c-33.118316-19.132632-59.984842-11.102316-59.984843 17.866105 0 28.995368 26.866526 68.069053 59.984843 87.174737-9.943579-17.973895-18.000842-41.957053-18.000843-62.922105 0-20.938105 8.057263-35.597474 18.000843-42.118737zM761.182316 603.162947c-27.055158-37.645474-60.685474-55.834947-74.994527-40.609684-14.336 15.225263-3.988211 58.125474 23.093895 95.770948-2.074947-17.731368 0.889263-37.025684 11.237053-48.047158 10.347789-10.994526 26.489263-11.964632 40.663579-7.114106z"
                    fill="#6D8ACA"
                />
                <path
                    d="M815.616 722.081684c-5.012211-40.690526-29.237895-81.785263-54.056421-91.728842-24.818526-9.916632-40.879158 15.063579-35.866947 55.754105 8.973474-8.003368 24.306526-11.317895 42.226526-4.122947 17.946947 7.168 35.705263 23.686737 47.696842 40.097684z"
                    fill="#6D8ACA"
                />
                <path
                    d="M789.018947 826.421895c26.462316-7.68 34.654316-42.010947 18.297264-76.584421-16.357053-34.573474-51.119158-56.400842-77.581474-48.693895 14.848 12.314947 32.202105 33.468632 44.032 58.448842 11.802947 25.007158 16.276211 49.906526 15.25221 66.829474zM691.900632 824.912842c31.636211 29.534316 64.053895 32.202105 72.353684 5.955369 8.326737-26.273684-10.617263-71.572211-42.253474-101.133474 5.982316 19.968 8.138105 44.678737 2.128842 63.649684-6.009263 18.970947-19.240421 29.291789-32.229052 31.528421z"
                    fill="#6D8ACA"
                />
                <path
                    d="M615.666526 746.954105c19.725474 42.091789 52.655158 71.518316 73.458527 65.697684 20.776421-5.820632 21.638737-44.732632 1.913263-86.824421-2.883368 15.090526-11.937684 29.210947-26.974316 33.441685-15.036632 4.203789-33.684211-2.155789-48.397474-12.314948z"
                    fill="#6D8ACA"
                />
                <path
                    d="M574.733474 652.207158c3.098947 38.965895 25.923368 80.114526 50.984421 91.809684 25.061053 11.695158 42.873263-10.455579 39.801263-49.448421-9.674105 6.763789-25.6 8.461474-43.708632 0-18.108632-8.434526-35.543579-25.734737-47.077052-42.361263z"
                    fill="#6D8ACA"
                />
                <path
                    d="M576.323368 567.430737c-12.207158 27.540211-3.260632 69.658947 19.941053 93.992421 23.201684 24.306526 51.981474 21.692632 64.188632-5.874526-13.473684-2.021053-31.932632-11.506526-48.693895-29.076211a165.834105 165.834105 0 0 1-35.43579-59.041684z"
                    fill="#6D8ACA"
                />
                <path
                    d="M275.240421 76.665263L192.565895 28.941474v83.482947l82.674526 47.750737V76.665263z"
                    fill="#283961"
                />
                <path d="M275.240421 47.750737L192.565895 0v39.262316l82.674526 47.723789V47.750737z" fill="#E2B36B" />
                <path
                    d="M203.829895 6.494316L199.572211 4.042105v39.262316l4.257684 2.452211V6.494316zM214.016 12.395789L209.758316 9.943579v39.235368l4.257684 2.479158V12.395789zM224.229053 18.297263L219.971368 15.818105v39.262316l4.257685 2.452211V18.297263zM234.442105 24.171789L230.157474 21.719579v39.262316l4.284631 2.45221V24.171789zM244.628211 30.073263l-4.257685-2.479158v39.262316l4.257685 2.479158V30.073263zM254.841263 35.947789l-4.257684-2.45221V72.757895l4.257684 2.45221V35.947789zM265.054316 41.849263l-4.284632-2.45221v39.262315l4.284632 2.452211V41.849263zM275.240421 47.750737l-4.257684-2.479158v39.262316l4.257684 2.45221V47.750737z"
                    fill="#E1CBA9"
                />
                <path
                    d="M287.285895 113.825684l-22.743579-13.150316v53.328843l22.743579 13.123368V113.825684zM688.693895 312.562526L287.285895 80.788211v86.339368l401.408 231.747368v-86.312421z"
                    fill="#283961"
                />
                <path
                    d="M688.693895 285.130105L287.285895 53.355789v39.235369l401.408 231.774316v-39.235369z"
                    fill="#E2B36B"
                />
                <path
                    d="M291.543579 55.834947l-4.257684-2.479158v39.262316l4.257684 2.479158V55.834947zM303.966316 63.002947l-4.257684-2.479158v39.262316l4.257684 2.452211V63.002947zM316.362105 70.144l-4.257684-2.452211v39.262316l4.257684 2.452211V70.144zM328.784842 77.312l-4.257684-2.452211v39.262316l4.257684 2.452211V77.312zM341.207579 84.48l-4.284632-2.452211v39.262316l4.284632 2.452211V84.48zM353.603368 91.648l-4.257684-2.452211v39.262316l4.257684 2.452211V91.648zM366.026105 98.816l-4.284631-2.452211v39.262316l4.284631 2.452211V98.816zM378.421895 105.984l-4.257684-2.452211v39.235369l4.257684 2.479158V105.984zM390.844632 113.152l-4.257685-2.452211v39.235369l4.257685 2.479158V113.152zM403.240421 120.32l-4.257684-2.479158V157.103158l4.257684 2.479158V120.32zM415.663158 127.488l-4.257684-2.479158v39.262316l4.257684 2.479158V127.488zM428.058947 134.656l-4.257684-2.479158v39.262316l4.257684 2.45221V134.656zM440.481684 141.824l-4.257684-2.479158v39.262316l4.257684 2.45221V141.824zM452.904421 148.965053l-4.284632-2.452211v39.262316l4.284632 2.45221V148.965053zM465.300211 156.133053l-4.257685-2.452211V192.943158l4.257685 2.45221V156.133053zM477.722947 163.301053l-4.284631-2.452211v39.262316l4.284631 2.45221V163.301053zM490.118737 170.469053l-4.257684-2.452211v39.262316l4.257684 2.45221V170.469053zM502.541474 177.637053l-4.257685-2.452211v39.262316l4.257685 2.45221V177.637053zM514.937263 184.805053l-4.257684-2.452211v39.235369l4.257684 2.479157V184.805053zM527.36 191.973053l-4.257684-2.452211v39.235369l4.257684 2.479157V191.973053zM539.782737 199.141053l-4.284632-2.479158v39.262316l4.284632 2.479157V199.141053zM552.178526 206.309053l-4.257684-2.479158v39.262316l4.257684 2.479157V206.309053zM564.601263 213.477053l-4.284631-2.479158v39.262316l4.284631 2.45221V213.477053zM576.997053 220.645053l-4.257685-2.479158v39.262316l4.257685 2.45221V220.645053zM589.419789 227.786105l-4.257684-2.45221v39.262316l4.257684 2.45221V227.786105zM601.815579 234.954105l-4.257684-2.45221v39.262316l4.257684 2.45221V234.954105zM614.238316 242.122105l-4.257684-2.45221v39.262316l4.257684 2.45221V242.122105zM626.634105 249.290105l-4.257684-2.45221v39.262316l4.257684 2.45221V249.290105zM639.056842 256.458105l-4.257684-2.45221v39.262316l4.257684 2.45221V256.458105zM651.479579 263.626105l-4.284632-2.45221v39.235368l4.284632 2.479158V263.626105zM663.875368 270.794105l-4.257684-2.45221v39.235368l4.257684 2.479158v-39.262316zM676.298105 277.962105l-4.284631-2.479158v39.262316l4.284631 2.479158v-39.262316zM688.693895 285.130105l-4.257684-2.479158v39.262316l4.257684 2.479158v-39.262316z"
                    fill="#E1CBA9"
                />
                <path
                    d="M688.693895 285.130105s7.868632 103.235368 8.838737 103.235369c0.943158 0-8.838737 10.509474-8.838737 10.509473"
                    fill="#283961"
                />
                <path
                    d="M688.693895 909.635368L229.268211 644.365474v15.144421l459.425684 265.269894v-15.144421z"
                    fill="#6D8ACA"
                />
                <path
                    d="M760.589474 405.854316l-53.706106 31.016421 10.213053 5.901474 53.706105-31.016422-10.213052-5.901473zM787.294316 421.295158l-53.679158 30.989474 10.213053 5.901473 53.706105-30.989473-10.24-5.901474zM814.026105 436.736l-53.706105 30.989474 10.24 5.901473 53.679158-31.016421-10.213053-5.874526zM840.757895 452.149895l-53.706106 31.016421 10.213053 5.874526 53.706105-30.989474-10.213052-5.901473zM867.462737 467.590737l-53.679158 30.989474 10.213053 5.901473 53.706105-30.989473-10.24-5.901474zM894.194526 483.004632l-53.706105 31.016421 10.24 5.874526 53.679158-30.989474-10.213053-5.901473zM950.676211 594.917053l-58.233264 33.630315v13.797053l58.233264-33.603368v-13.824zM950.784 628.197053l-58.233263 33.630315v13.797053l58.233263-33.630316v-13.797052zM950.918737 661.477053l-58.260211 33.630315v13.797053l58.260211-33.630316v-13.797052zM951.026526 694.757053L892.766316 728.387368v13.797053l58.26021-33.630316v-13.797052zM951.134316 728.037053l-58.260211 33.630315v13.797053l58.260211-33.630316v-13.797052zM951.242105 761.317053L892.981895 794.947368v13.797053l58.26021-33.630316v-13.797052zM951.349895 794.597053l-58.260211 33.630315v13.797053l58.260211-33.630316v-13.797052zM951.457684 827.877053l-58.26021 33.630315v13.797053l58.26021-33.630316v-13.797052zM951.565474 861.157053l-58.260211 33.630315v13.797053l58.260211-33.630316v-13.797052zM951.673263 894.437053l-58.26021 33.603368v13.824l58.26021-33.630316v-13.797052z"
                    fill="#283961"
                />
            </svg>
        );
    }
}