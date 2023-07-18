#include <iostream>
#include <fstream>
#include <string>
#include <string_view>
#include <unordered_map>
#include <unordered_set>
#include <vector>

#include "absl/flags/flag.h"
#include "absl/flags/parse.h"
#include "absl/strings/str_format.h"
#include "httplib.h"
#include "include/nlohmann/json.hpp"

ABSL_FLAG(int32_t, port, 6501, "Port to listen on.");
ABSL_FLAG(std::string, word_list_path, "", "List of candidate words. One word per line.");

const int word_len = 5;

using WordSet = std::unordered_set<std::string>;
using WordSetView = std::unordered_set<std::string_view>;

using CharMap = std::unordered_map<char, WordSetView>;
using PositionMap = std::vector<CharMap>;

PositionMap GetPositionMap(const WordSet& words){
  PositionMap pos_map(word_len);

  for(const auto& word: words){
    for(int i = 0; i<word.size(); i++){
      pos_map[i][word[i]].insert(word);
    }
  }

  return pos_map;
}

CharMap GetCharMap(const WordSet& words){
  CharMap char_map;

  for(const auto& word: words){
    for(const auto& c: word){
      char_map[c].insert(word);
    }
  }

  return char_map;
}

WordSet ReadWords(const std::string& path){
  WordSet word_set;
  std::ifstream file(path, std::ifstream::in);

  if(!file.is_open())
    return word_set;

  std::string line;
  while(file){
    std::getline(file, line);
    word_set.insert(std::move(line));
  }

  return word_set;
}

// Destructively mutates input sets.
WordSetView* intersect(WordSetView& s1, WordSetView& s2){
  WordSetView* smaller = &s1;
  WordSetView* bigger = &s2;

  if(s1.size() > s2.size()){
    smaller = &s2;
    bigger = &s1;
  }

  for(auto it = smaller->begin(); it != smaller->end();){
    if(bigger->find(*it) == bigger->end())
      it = smaller->erase(it);
    else
      it++;
  }

  return smaller;
}

class Choices {
  public:
    Choices(
      std::unordered_map<int, char> known_in_position,
      std::vector<char> known_out_of_position,
      std::vector<char> known_bad) :
      known_in_position_(std::move(known_in_position)),
      known_out_of_position_(std::move(known_out_of_position)),
      known_bad_(std::move(known_bad)){
      }
    Choices() = default;
    ~Choices() = default;

  std::unordered_map<int, char> known_in_position_;
  std::vector<char> known_out_of_position_;
  std::vector<char> known_bad_;
};

class State {
 public:
  State() = default;
  State(
    Choices* choices,
    PositionMap pos_map,
    CharMap char_map,
    const WordSet* word_set) :
    choices_(choices),
    pos_map_(std::move(pos_map)),
    char_map_(std::move(char_map)),
    word_set_(word_set)
  {
  }
  ~State() = default;

  Choices* choices_;
  PositionMap pos_map_;
  CharMap char_map_;
  const WordSet* word_set_;

  // Note that this approach can still suggest impossible words. The position
  // where the known out of position letters were discovered is not taken into
  // account. In other words, if a letter is 'yellow' in position 0, we'd want
  // to only consider words where that letter is found in positions 1-4. The
  // current approach, using the any map, will still consider position 0.
  // The various maps in this data structure are mutated and are no longer
  // truthful to input data.
  std::vector<std::string_view> SuggestWords() {
    std::vector<std::string_view> results;

    // Start with every word as a candidate.
    // Deep copy because the view gets mutated.
    WordSetView view;
    for(const auto& word : *word_set_)
      view.insert(word);

    WordSetView* candidates = &view;

    // Consider the words matching the known positions, and intersect those.
    for(int i = 0; i < word_len; i++){
      auto found = choices_->known_in_position_.find(i);
      if(found == choices_->known_in_position_.end())
        continue;

      candidates = intersect(*candidates, pos_map_[i][found->second]);
    }

    // Consider the words matching out of positions, and intersect those.
    for(const auto& c : choices_->known_out_of_position_){
      candidates = intersect(*candidates, char_map_[c]);
    }

    // Filter out impossible words.
    for(const auto& c : choices_->known_bad_){
      for(const auto& word : char_map_[c]){
        candidates->erase(word);
      }
    }

    results.reserve(candidates->size());
    for(const auto& word : *candidates){
      // Don't need to use std::move here as string views are very lightweight.
      results.push_back(word);
    }

    return results;
  }
};

std::vector<std::string_view> SuggestWords(const WordSet* word_set, Choices* choices) {
    // Everything else is just a view into the strings.
    PositionMap pos_map = GetPositionMap(*word_set);
    CharMap char_map = GetCharMap(*word_set);

    State state(
        choices,
        std::move(pos_map),
        std::move(char_map),
        word_set
    );

    return state.SuggestWords();
}

struct WireChoices {
  std::unordered_map<std::string, std::string> known_in_position;
  std::string known_out_of_position;
  std::string known_bad;
};

int main(int argc, char* argv[]){
  absl::ParseCommandLine(argc, argv);

  using json = nlohmann::json;

  // The original set of words. This must live for the lifetime of the program.
  std::string word_list_path = absl::GetFlag(FLAGS_word_list_path);
  WordSet word_set = ReadWords(word_list_path);

  httplib::Server svr;
  svr.Options("/suggest", [](const httplib::Request& req, httplib::Response& res) {
    res.set_header("Access-Control-Allow-Origin", "*");
    res.set_header("Access-Control-Allow-Methods", "POST");
    res.set_header("Access-Control-Allow-Headers", "Content-Type");
    res.set_header("Access-Control-Max-Age", "3600");
  });
  svr.Post("/suggest", [&word_set](const httplib::Request& req, httplib::Response& res) {
    res.set_header("Access-Control-Allow-Origin", "*");
    json data = json::parse(req.body);

    WireChoices wire_choices;
    try {
      wire_choices = WireChoices{
        data.at("KnownInPosition").get<std::unordered_map<std::string, std::string>>(),
        data.at("KnownOutOfPosition").get<std::string>(),
        data.at("KnownBad").get<std::string>()
      };
    } catch(json::type_error e){
			auto help = R"(Failed to unmarshal the application/json request. Expected Schema:
	"KnownInPosition": map[int]string
	"KnownOutOfPosition": string
	"KnownBad": string
)";
      res.status = 400;
      res.set_content(help, "plain/text");
      return;
    }

    // Convert wire data to application data.
    std::unordered_map<int, char> known_in_position;
    std::vector<char> known_out_of_position;
    std::vector<char> known_bad;
    std::unordered_set<char> known_good;

    for(const auto& [k, v] : wire_choices.known_in_position){
      known_in_position[std::stoi(k)] = v[0];
      known_good.insert(v[0]);
    }

    for(const auto& s: wire_choices.known_out_of_position){
      known_out_of_position.push_back(s);
      known_good.insert(s);
    }

    for(const auto& s: wire_choices.known_bad){
      known_bad.push_back(s);
      if(known_good.count(s) == 1){
        res.set_content(absl::StrFormat("Conflicting configuration with letter '%c'.", s), "plain/text");
        return;
      }
    }

    Choices choices(
      std::move(known_in_position),
      std::move(known_out_of_position),
      std::move(known_bad)
    );

    auto results = SuggestWords(&word_set, &choices);
    if(results.empty()){
      res.set_content("Unable to find any words.", "plain/text");
      return;
    }
    json response = results;

    res.set_content(response.dump(), "application/json");
  });

  int32_t httpEndPoint = absl::GetFlag(FLAGS_port);
  std::cout << absl::StrFormat("Starting C++ server on :%d", httpEndPoint) << std::endl;
  // Listen on INADDR6_ANY.
  if(!svr.listen("::", httpEndPoint))
    std::cout << "Failed to listen." << std::endl;

  return 0;
}
