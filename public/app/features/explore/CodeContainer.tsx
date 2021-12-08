import React, { PureComponent } from 'react';
import { connect, ConnectedProps } from 'react-redux';
import { splitOpen } from './state/main';
import { ExploreId, ExploreItemState } from 'app/types/explore';
import { StoreState } from 'app/types';
import SyntaxHighlighter from 'react-syntax-highlighter';
import { DataFrame } from '@grafana/data';

interface CodeContainerProps extends PropsFromRedux {
  dataFrames: DataFrame[];
  exploreId: ExploreId;
}

function mapStateToProps(state: StoreState /*, { exploreId }: CodeContainerProps*/) {
  console.log('in mapStateToProps');
  // const explore = state.explore;
  // // @ts-ignore
  // const item: ExploreItemState = explore[exploreId];
  // const { dataFrames } = item;
  // console.log("item");
  // console.log(item);
  // console.log("dataFrames");
  // console.log(dataFrames);

  // const loading = tableResult && tableResult.length > 0 ? false : loadingInState;
  // return { loading, tableResult, range };

  return {};
  // return { dataFrames };
}

const mapDispatchToProps = {
  splitOpen,
};

const connector = connect(mapStateToProps /*, mapDispatchToProps*/);

//type Props = CodeContainerProps & ConnectedProps<typeof connector>;

export class CodeContainer extends PureComponent<CodeContainerProps> {
  render() {
    const { dataFrames } = this.props;
    console.log('in render');
    console.log(dataFrames);

    console.log(Object.keys(dataFrames));
    // for (const f of dataFrames.fields) {
    //   console.log('name', f.name);
    // }

    return (
      <SyntaxHighlighter language="javascript" wrapLines="true" showLineNumbers="true">
        this.code();
      </SyntaxHighlighter>
    );
  }
}

export default connector(CodeContainer);
type PropsFromRedux = ConnectedProps<typeof connector>;
