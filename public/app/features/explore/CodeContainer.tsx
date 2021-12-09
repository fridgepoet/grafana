import React, { PureComponent } from 'react';
import { connect, ConnectedProps } from 'react-redux';
import { splitOpen } from './state/main';
import { ExploreId, ExploreItemState } from 'app/types/explore';
import { StoreState } from 'app/types';
import SyntaxHighlighter from 'react-syntax-highlighter';

interface CodeContainerProps {
  exploreId: ExploreId;
}

function mapStateToProps(state: StoreState, { exploreId }: CodeContainerProps) {
  const explore = state.explore;

  // @ts-ignore
  const item: ExploreItemState = explore[exploreId];
  const { queryResponse } = item;

  const series = queryResponse.series.filter((series) => {
    return series.meta?.custom?.Code === true;
  });

  return { series: series };
}

const mapDispatchToProps = {
  splitOpen,
};

export class CodeContainer extends PureComponent<Props> {
  render() {
    console.log('props');
    console.log(this.props);

    const { series } = this.props;

    for (const s of series) {
      console.log('single series');
      console.log(s);
    }

    const codeField = series[0].fields[0];
    console.log('field name', codeField.name);

    const codeValue = codeField.values.toArray()[0];
    console.log('codeValue');
    console.log(codeValue);

    const language = 'python';

    return (
      <div>
        <SyntaxHighlighter language={language} wrapLines={true} showLineNumbers={true}>
          {String(codeValue)}
        </SyntaxHighlighter>
      </div>
    );
  }
}

const connector = connect(mapStateToProps, mapDispatchToProps);
type Props = CodeContainerProps & ConnectedProps<typeof connector>;
export default connector(CodeContainer);
